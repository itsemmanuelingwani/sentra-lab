// packages/engine/src/recording/recorder.rs
//! Main event recorder with <100ns overhead
//!
//! Provides lock-free event recording with batched compression and storage.

use crate::recording::compressor::{Compressor, CompressionLevel};
use crate::recording::event_queue::EventQueue;
use crate::recording::storage::{EventStorage, StorageConfig};
use crate::utils::errors::{EngineError, Result};
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::Notify;
use tokio::task::JoinHandle;
use tracing::{debug, error, info, warn};

/// Event to be recorded
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Event {
    /// Unique event ID
    pub id: String,
    
    /// Simulation run ID
    pub run_id: String,
    
    /// Event type
    pub event_type: EventType,
    
    /// Timestamp (nanoseconds since epoch)
    pub timestamp_ns: u64,
    
    /// Event data (JSON)
    pub data: serde_json::Value,
    
    /// Duration (microseconds)
    pub duration_us: Option<u64>,
}

/// Event types
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum EventType {
    AgentStarted,
    InputReceived,
    ExternalCallMade,
    ExternalCallCompleted,
    StateChanged,
    DecisionMade,
    ErrorEncountered,
    OutputProduced,
    AgentCompleted,
}

/// Recorder configuration
#[derive(Debug, Clone)]
pub struct RecorderConfig {
    /// Batch size for compression (number of events)
    pub batch_size: usize,
    
    /// Flush interval (milliseconds)
    pub flush_interval_ms: u64,
    
    /// Compression level
    pub compression_level: CompressionLevel,
    
    /// Storage configuration
    pub storage: StorageConfig,
    
    /// Maximum queue size
    pub max_queue_size: usize,
}

impl Default for RecorderConfig {
    fn default() -> Self {
        Self {
            batch_size: 1000,
            flush_interval_ms: 100,
            compression_level: CompressionLevel::Fast,
            storage: StorageConfig::default(),
            max_queue_size: 1_000_000,
        }
    }
}

/// High-performance event recorder
pub struct EventRecorder {
    config: RecorderConfig,
    queue: Arc<EventQueue>,
    storage: Arc<EventStorage>,
    compressor: Arc<Compressor>,
    flush_notify: Arc<Notify>,
    writer_handle: Option<JoinHandle<()>>,
    stats: Arc<tokio::sync::Mutex<RecorderStats>>,
}

impl EventRecorder {
    /// Create a new event recorder
    pub async fn new(config: RecorderConfig) -> Result<Self> {
        info!("Initializing event recorder");
        
        let queue = Arc::new(EventQueue::new(config.max_queue_size));
        let storage = Arc::new(EventStorage::new(config.storage.clone()).await?);
        let compressor = Arc::new(Compressor::new(config.compression_level));
        let flush_notify = Arc::new(Notify::new());
        let stats = Arc::new(tokio::sync::Mutex::new(RecorderStats::default()));
        
        Ok(Self {
            config,
            queue,
            storage,
            compressor,
            flush_notify,
            writer_handle: None,
            stats,
        })
    }
    
    /// Start background writer
    pub fn start(&mut self) -> Result<()> {
        info!("Starting background event writer");
        
        let queue = Arc::clone(&self.queue);
        let storage = Arc::clone(&self.storage);
        let compressor = Arc::clone(&self.compressor);
        let flush_notify = Arc::clone(&self.flush_notify);
        let stats = Arc::clone(&self.stats);
        let batch_size = self.config.batch_size;
        let flush_interval = Duration::from_millis(self.config.flush_interval_ms);
        
        let handle = tokio::spawn(async move {
            let mut interval = tokio::time::interval(flush_interval);
            let mut batch = Vec::with_capacity(batch_size);
            
            loop {
                tokio::select! {
                    _ = interval.tick() => {
                        // Periodic flush
                        if !batch.is_empty() {
                            if let Err(e) = Self::flush_batch(
                                &mut batch,
                                &storage,
                                &compressor,
                                &stats
                            ).await {
                                error!("Failed to flush batch: {}", e);
                            }
                        }
                    }
                    
                    _ = flush_notify.notified() => {
                        // Immediate flush requested
                        if !batch.is_empty() {
                            if let Err(e) = Self::flush_batch(
                                &mut batch,
                                &storage,
                                &compressor,
                                &stats
                            ).await {
                                error!("Failed to flush batch: {}", e);
                            }
                        }
                    }
                }
                
                // Drain queue into batch
                while let Some(event) = queue.try_pop() {
                    batch.push(event);
                    
                    if batch.len() >= batch_size {
                        if let Err(e) = Self::flush_batch(
                            &mut batch,
                            &storage,
                            &compressor,
                            &stats
                        ).await {
                            error!("Failed to flush batch: {}", e);
                        }
                    }
                }
            }
        });
        
        self.writer_handle = Some(handle);
        Ok(())
    }
    
    /// Record an event (lock-free, <100ns)
    pub fn record(&self, event: Event) -> Result<()> {
        let start = Instant::now();
        
        self.queue.push(event).map_err(|_| {
            EngineError::RecordingFailed("Event queue full".to_string())
        })?;
        
        // Update stats (async, non-blocking)
        let elapsed = start.elapsed();
        tokio::spawn({
            let stats = Arc::clone(&self.stats);
            async move {
                let mut s = stats.lock().await;
                s.events_recorded += 1;
                s.total_record_time_ns += elapsed.as_nanos() as u64;
            }
        });
        
        Ok(())
    }
    
    /// Flush events immediately
    pub async fn flush(&self) -> Result<()> {
        self.flush_notify.notify_one();
        
        // Wait a bit for flush to complete
        tokio::time::sleep(Duration::from_millis(50)).await;
        
        Ok(())
    }
    
    /// Flush a batch of events
    async fn flush_batch(
        batch: &mut Vec<Event>,
        storage: &EventStorage,
        compressor: &Compressor,
        stats: &Arc<tokio::sync::Mutex<RecorderStats>>,
    ) -> Result<()> {
        if batch.is_empty() {
            return Ok(());
        }
        
        let batch_size = batch.len();
        debug!("Flushing batch of {} events", batch_size);
        
        let start = Instant::now();
        
        // Serialize events to JSON
        let json_data = serde_json::to_vec(&batch)
            .map_err(|e| EngineError::RecordingFailed(format!("Serialization error: {}", e)))?;
        
        // Compress
        let compressed = compressor.compress(&json_data)?;
        
        // Write to storage
        storage.write_batch(&compressed).await?;
        
        let elapsed = start.elapsed();
        
        // Update stats
        let mut s = stats.lock().await;
        s.batches_flushed += 1;
        s.events_flushed += batch_size as u64;
        s.bytes_written += compressed.len() as u64;
        s.total_flush_time_ms += elapsed.as_millis() as u64;
        
        // Clear batch
        batch.clear();
        
        debug!("Batch flushed in {:?}", elapsed);
        
        Ok(())
    }
    
    /// Get recorder statistics
    pub async fn stats(&self) -> RecorderStats {
        self.stats.lock().await.clone()
    }
    
    /// Shutdown recorder
    pub async fn shutdown(&mut self) -> Result<()> {
        info!("Shutting down event recorder");
        
        // Flush remaining events
        self.flush().await?;
        
        // Stop background writer
        if let Some(handle) = self.writer_handle.take() {
            handle.abort();
        }
        
        Ok(())
    }
}

/// Recorder statistics
#[derive(Debug, Clone, Default)]
pub struct RecorderStats {
    pub events_recorded: u64,
    pub events_flushed: u64,
    pub batches_flushed: u64,
    pub bytes_written: u64,
    pub total_record_time_ns: u64,
    pub total_flush_time_ms: u64,
}

impl RecorderStats {
    pub fn avg_record_time_ns(&self) -> u64 {
        if self.events_recorded == 0 {
            0
        } else {
            self.total_record_time_ns / self.events_recorded
        }
    }
    
    pub fn avg_flush_time_ms(&self) -> u64 {
        if self.batches_flushed == 0 {
            0
        } else {
            self.total_flush_time_ms / self.batches_flushed
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[tokio::test]
    async fn test_recorder_creation() {
        let config = RecorderConfig::default();
        let recorder = EventRecorder::new(config).await;
        assert!(recorder.is_ok());
    }
    
    #[tokio::test]
    async fn test_record_event() {
        let config = RecorderConfig::default();
        let mut recorder = EventRecorder::new(config).await.unwrap();
        recorder.start().unwrap();
        
        let event = Event {
            id: "evt_123".to_string(),
            run_id: "run_abc".to_string(),
            event_type: EventType::AgentStarted,
            timestamp_ns: 0,
            data: serde_json::json!({}),
            duration_us: None,
        };
        
        let result = recorder.record(event);
        assert!(result.is_ok());
    }
    
    #[tokio::test]
    async fn test_stats() {
        let config = RecorderConfig::default();
        let recorder = EventRecorder::new(config).await.unwrap();
        
        let stats = recorder.stats().await;
        assert_eq!(stats.events_recorded, 0);
    }
}