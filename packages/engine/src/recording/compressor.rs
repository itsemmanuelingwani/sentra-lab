// packages/engine/src/recording/compressor.rs
//! zstd batch compression for event data
//!
//! Provides fast compression with good compression ratios (10:1 typical).

use crate::utils::errors::{EngineError, Result};
use tracing::debug;

/// Compression levels
#[derive(Debug, Clone, Copy)]
pub enum CompressionLevel {
    /// Fast compression (level 1)
    Fast,
    
    /// Balanced (level 3)
    Balanced,
    
    /// Best compression (level 19)
    Best,
}

impl CompressionLevel {
    pub fn as_i32(&self) -> i32 {
        match self {
            CompressionLevel::Fast => 1,
            CompressionLevel::Balanced => 3,
            CompressionLevel::Best => 19,
        }
    }
}

/// Compressor using zstd
pub struct Compressor {
    level: CompressionLevel,
}

impl Compressor {
    /// Create a new compressor
    pub fn new(level: CompressionLevel) -> Self {
        Self { level }
    }
    
    /// Compress data
    pub fn compress(&self, data: &[u8]) -> Result<Vec<u8>> {
        let level = self.level.as_i32();
        
        debug!("Compressing {} bytes at level {}", data.len(), level);
        
        let compressed = zstd::encode_all(data, level).map_err(|e| {
            EngineError::CompressionFailed(format!("Compression error: {}", e))
        })?;
        
        let ratio = data.len() as f64 / compressed.len() as f64;
        debug!(
            "Compressed {} bytes -> {} bytes (ratio: {:.2}x)",
            data.len(),
            compressed.len(),
            ratio
        );
        
        Ok(compressed)
    }
    
    /// Decompress data
    pub fn decompress(&self, data: &[u8]) -> Result<Vec<u8>> {
        debug!("Decompressing {} bytes", data.len());
        
        let decompressed = zstd::decode_all(data).map_err(|e| {
            EngineError::CompressionFailed(format!("Decompression error: {}", e))
        })?;
        
        debug!(
            "Decompressed {} bytes -> {} bytes",
            data.len(),
            decompressed.len()
        );
        
        Ok(decompressed)
    }
    
    /// Estimate compressed size (approximate)
    pub fn estimate_compressed_size(&self, data: &[u8]) -> usize {
        // Rough estimate based on typical compression ratios
        match self.level {
            CompressionLevel::Fast => data.len() / 5,      // ~5x compression
            CompressionLevel::Balanced => data.len() / 10, // ~10x compression
            CompressionLevel::Best => data.len() / 15,     // ~15x compression
        }
    }
}

impl Default for Compressor {
    fn default() -> Self {
        Self::new(CompressionLevel::Balanced)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_compression_levels() {
        assert_eq!(CompressionLevel::Fast.as_i32(), 1);
        assert_eq!(CompressionLevel::Balanced.as_i32(), 3);
        assert_eq!(CompressionLevel::Best.as_i32(), 19);
    }
    
    #[test]
    fn test_compress_decompress() {
        let compressor = Compressor::new(CompressionLevel::Balanced);
        
        let data = b"Hello, World! This is test data.".repeat(100);
        
        let compressed = compressor.compress(&data).unwrap();
        assert!(compressed.len() < data.len());
        
        let decompressed = compressor.decompress(&compressed).unwrap();
        assert_eq!(decompressed, data);
    }
    
    #[test]
    fn test_json_compression() {
        let compressor = Compressor::new(CompressionLevel::Balanced);
        
        // Simulate JSON event data
        let json_data = r#"{"id":"evt_123","type":"agent_started","data":{}}"#.repeat(1000);
        
        let compressed = compressor.compress(json_data.as_bytes()).unwrap();
        
        let ratio = json_data.len() as f64 / compressed.len() as f64;
        assert!(ratio > 5.0); // Should achieve at least 5x compression
    }
    
    #[test]
    fn test_compression_levels_comparison() {
        let data = b"Test data for compression".repeat(100);
        
        let fast = Compressor::new(CompressionLevel::Fast);
        let balanced = Compressor::new(CompressionLevel::Balanced);
        let best = Compressor::new(CompressionLevel::Best);
        
        let fast_size = fast.compress(&data).unwrap().len();
        let balanced_size = balanced.compress(&data).unwrap().len();
        let best_size = best.compress(&data).unwrap().len();
        
        // Best should compress more than fast
        assert!(best_size <= balanced_size);
        assert!(balanced_size <= fast_size);
    }
}