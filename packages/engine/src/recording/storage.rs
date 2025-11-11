// packages/engine/src/recording/storage.rs
//! Event storage using SQLite + file system
//!
//! Stores event metadata in SQLite and compressed event data in files.

use crate::utils::errors::{EngineError, Result};
use rusqlite::{params, Connection};
use std::path::PathBuf;
use std::sync::Arc;
use tokio::fs;
use tokio::sync::Mutex;
use tracing::{debug, info};

/// Storage configuration
#[derive(Debug, Clone)]
pub struct StorageConfig {
    /// Base directory for storage
    pub base_dir: PathBuf,
    
    /// SQLite database file name
    pub db_name: String,
    
    /// Events directory name
    pub events_dir: String,
}

impl Default for StorageConfig {
    fn default() -> Self {
        Self {
            base_dir: PathBuf::from("~/.sentra-lab/simulations"),
            db_name: "events.db".to_string(),
            events_dir: "events".to_string(),
        }
    }
}

/// Event storage
pub struct EventStorage {
    config: StorageConfig,
    db: Arc<Mutex<Connection>>,
    batch_counter: Arc<Mutex<u64>>,
}

impl EventStorage {
    /// Create a new event storage
    pub async fn new(config: StorageConfig) -> Result<Self> {
        // Create base directory
        fs::create_dir_all(&config.base_dir).await.map_err(|e| {
            EngineError::StorageFailed(format!("Failed to create directory: {}", e))
        })?;
        
        // Create events directory
        let events_dir = config.base_dir.join(&config.events_dir);
        fs::create_dir_all(&events_dir).await.map_err(|e| {
            EngineError::StorageFailed(format!("Failed to create events directory: {}", e))
        })?;
        
        // Open SQLite database
        let db_path = config.base_dir.join(&config.db_name);
        let conn = Connection::open(&db_path).map_err(|e| {
            EngineError::StorageFailed(format!("Failed to open database: {}", e))
        })?;
        
        let storage = Self {
            config,
            db: Arc::new(Mutex::new(conn)),
            batch_counter: Arc::new(Mutex::new(0)),
        };
        
        // Initialize schema
        storage.init_schema().await?;
        
        info!("Event storage initialized at {:?}", storage.config.base_dir);
        
        Ok(storage)
    }
    
    /// Initialize database schema
    async fn init_schema(&self) -> Result<()> {
        let db = self.db.lock().await;
        
        db.execute(
            r#"
            CREATE TABLE IF NOT EXISTS event_batches (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                batch_id TEXT NOT NULL,
                file_path TEXT NOT NULL,
                event_count INTEGER NOT NULL,
                compressed_size INTEGER NOT NULL,
                created_at INTEGER NOT NULL
            )
            "#,
            [],
        )
        .map_err(|e| EngineError::StorageFailed(format!("Schema creation failed: {}", e)))?;
        
        db.execute(
            r#"
            CREATE INDEX IF NOT EXISTS idx_batch_id ON event_batches(batch_id)
            "#,
            [],
        )
        .map_err(|e| EngineError::StorageFailed(format!("Index creation failed: {}", e)))?;
        
        Ok(())
    }
    
    /// Write a compressed batch of events
    pub async fn write_batch(&self, compressed_data: &[u8]) -> Result<()> {
        // Generate batch ID
        let mut counter = self.batch_counter.lock().await;
        *counter += 1;
        let batch_id = format!("batch_{:08}", *counter);
        drop(counter);
        
        // Write compressed data to file
        let file_path = self.config.base_dir
            .join(&self.config.events_dir)
            .join(format!("{}.zst", batch_id));
        
        fs::write(&file_path, compressed_data).await.map_err(|e| {
            EngineError::StorageFailed(format!("Failed to write batch file: {}", e))
        })?;
        
        debug!("Wrote batch {} ({} bytes)", batch_id, compressed_data.len());
        
        // Record metadata in database
        let db = self.db.lock().await;
        db.execute(
            r#"
            INSERT INTO event_batches (batch_id, file_path, event_count, compressed_size, created_at)
            VALUES (?, ?, ?, ?, ?)
            "#,
            params![
                batch_id,
                file_path.to_string_lossy(),
                0, // TODO: Extract event count from batch
                compressed_data.len() as i64,
                chrono::Utc::now().timestamp(),
            ],
        )
        .map_err(|e| {
            EngineError::StorageFailed(format!("Failed to record batch metadata: {}", e))
        })?;
        
        Ok(())
    }
    
    /// Read a batch by ID
    pub async fn read_batch(&self, batch_id: &str) -> Result<Vec<u8>> {
        // Get file path from database
        let db = self.db.lock().await;
        let file_path: String = db
            .query_row(
                "SELECT file_path FROM event_batches WHERE batch_id = ?",
                params![batch_id],
                |row| row.get(0),
            )
            .map_err(|e| {
                EngineError::StorageFailed(format!("Batch not found: {}", e))
            })?;
        
        drop(db);
        
        // Read compressed data
        let data = fs::read(&file_path).await.map_err(|e| {
            EngineError::StorageFailed(format!("Failed to read batch file: {}", e))
        })?;
        
        Ok(data)
    }
    
    /// List all batches
    pub async fn list_batches(&self) -> Result<Vec<BatchMetadata>> {
        let db = self.db.lock().await;
        
        let mut stmt = db
            .prepare("SELECT batch_id, event_count, compressed_size, created_at FROM event_batches ORDER BY id")
            .map_err(|e| {
                EngineError::StorageFailed(format!("Query preparation failed: {}", e))
            })?;
        
        let batches = stmt
            .query_map([], |row| {
                Ok(BatchMetadata {
                    batch_id: row.get(0)?,
                    event_count: row.get(1)?,
                    compressed_size: row.get(2)?,
                    created_at: row.get(3)?,
                })
            })
            .map_err(|e| {
                EngineError::StorageFailed(format!("Query execution failed: {}", e))
            })?
            .collect::<std::result::Result<Vec<_>, _>>()
            .map_err(|e| {
                EngineError::StorageFailed(format!("Result collection failed: {}", e))
            })?;
        
        Ok(batches)
    }
    
    /// Get storage statistics
    pub async fn stats(&self) -> Result<StorageStats> {
        let db = self.db.lock().await;
        
        let total_batches: i64 = db
            .query_row("SELECT COUNT(*) FROM event_batches", [], |row| row.get(0))
            .unwrap_or(0);
        
        let total_size: i64 = db
            .query_row(
                "SELECT SUM(compressed_size) FROM event_batches",
                [],
                |row| row.get(0),
            )
            .unwrap_or(0);
        
        Ok(StorageStats {
            total_batches: total_batches as u64,
            total_size_bytes: total_size as u64,
        })
    }
}

/// Batch metadata
#[derive(Debug, Clone)]
pub struct BatchMetadata {
    pub batch_id: String,
    pub event_count: i64,
    pub compressed_size: i64,
    pub created_at: i64,
}

/// Storage statistics
#[derive(Debug, Clone)]
pub struct StorageStats {
    pub total_batches: u64,
    pub total_size_bytes: u64,
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;
    
    #[tokio::test]
    async fn test_storage_creation() {
        let dir = tempdir().unwrap();
        let config = StorageConfig {
            base_dir: dir.path().to_path_buf(),
            ..Default::default()
        };
        
        let storage = EventStorage::new(config).await;
        assert!(storage.is_ok());
    }
    
    #[tokio::test]
    async fn test_write_read_batch() {
        let dir = tempdir().unwrap();
        let config = StorageConfig {
            base_dir: dir.path().to_path_buf(),
            ..Default::default()
        };
        
        let storage = EventStorage::new(config).await.unwrap();
        
        let data = b"test compressed data";
        storage.write_batch(data).await.unwrap();
        
        let batches = storage.list_batches().await.unwrap();
        assert_eq!(batches.len(), 1);
        
        let batch_id = &batches[0].batch_id;
        let read_data = storage.read_batch(batch_id).await.unwrap();
        assert_eq!(read_data, data);
    }
    
    #[tokio::test]
    async fn test_storage_stats() {
        let dir = tempdir().unwrap();
        let config = StorageConfig {
            base_dir: dir.path().to_path_buf(),
            ..Default::default()
        };
        
        let storage = EventStorage::new(config).await.unwrap();
        
        storage.write_batch(b"data1").await.unwrap();
        storage.write_batch(b"data2").await.unwrap();
        
        let stats = storage.stats().await.unwrap();
        assert_eq!(stats.total_batches, 2);
    }
}