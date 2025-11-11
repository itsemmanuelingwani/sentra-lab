// packages/engine/src/recording/mmap_writer.rs
//! Memory-mapped file writer for zero-copy writes
//!
//! Provides high-performance file writes using memory-mapped I/O.

use crate::utils::errors::{EngineError, Result};
use memmap2::{MmapMut, MmapOptions};
use std::fs::{File, OpenOptions};
use std::io::Write;
use std::path::Path;
use tracing::{debug, warn};

/// Memory-mapped file writer
pub struct MmapWriter {
    file: File,
    mmap: Option<MmapMut>,
    position: usize,
    capacity: usize,
}

impl MmapWriter {
    /// Create a new memory-mapped writer
    pub fn new<P: AsRef<Path>>(path: P, capacity: usize) -> Result<Self> {
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .open(path.as_ref())
            .map_err(|e| EngineError::StorageFailed(format!("Failed to open file: {}", e)))?;
        
        // Set file size
        file.set_len(capacity as u64).map_err(|e| {
            EngineError::StorageFailed(format!("Failed to set file size: {}", e))
        })?;
        
        // Create memory map
        let mmap = unsafe {
            MmapOptions::new().map_mut(&file).map_err(|e| {
                EngineError::StorageFailed(format!("Failed to create memory map: {}", e))
            })?
        };
        
        debug!("Created memory-mapped file with capacity {} bytes", capacity);
        
        Ok(Self {
            file,
            mmap: Some(mmap),
            position: 0,
            capacity,
        })
    }
    
    /// Write data to memory-mapped file
    pub fn write(&mut self, data: &[u8]) -> Result<usize> {
        if self.position + data.len() > self.capacity {
            // Need to grow the file
            self.grow(data.len())?;
        }
        
        if let Some(ref mut mmap) = self.mmap {
            let len = data.len();
            mmap[self.position..self.position + len].copy_from_slice(data);
            self.position += len;
            
            Ok(len)
        } else {
            Err(EngineError::StorageFailed(
                "Memory map not available".to_string(),
            ))
        }
    }
    
    /// Flush changes to disk
    pub fn flush(&mut self) -> Result<()> {
        if let Some(ref mut mmap) = self.mmap {
            mmap.flush().map_err(|e| {
                EngineError::StorageFailed(format!("Failed to flush memory map: {}", e))
            })?;
        }
        
        self.file.sync_all().map_err(|e| {
            EngineError::StorageFailed(format!("Failed to sync file: {}", e))
        })?;
        
        Ok(())
    }
    
    /// Grow the memory-mapped file
    fn grow(&mut self, additional: usize) -> Result<()> {
        warn!("Growing memory-mapped file by {} bytes", additional);
        
        // Unmap current mapping
        self.mmap = None;
        
        // Grow file
        let new_capacity = self.capacity + additional.max(self.capacity);
        self.file.set_len(new_capacity as u64).map_err(|e| {
            EngineError::StorageFailed(format!("Failed to grow file: {}", e))
        })?;
        
        // Remap
        let mmap = unsafe {
            MmapOptions::new().map_mut(&self.file).map_err(|e| {
                EngineError::StorageFailed(format!("Failed to remap file: {}", e))
            })?
        };
        
        self.mmap = Some(mmap);
        self.capacity = new_capacity;
        
        debug!("Memory-mapped file grown to {} bytes", new_capacity);
        
        Ok(())
    }
    
    /// Get current position
    pub fn position(&self) -> usize {
        self.position
    }
    
    /// Get capacity
    pub fn capacity(&self) -> usize {
        self.capacity
    }
    
    /// Get available space
    pub fn available(&self) -> usize {
        self.capacity - self.position
    }
}

impl Drop for MmapWriter {
    fn drop(&mut self) {
        // Flush on drop
        let _ = self.flush();
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::NamedTempFile;
    
    #[test]
    fn test_mmap_writer_creation() {
        let file = NamedTempFile::new().unwrap();
        let writer = MmapWriter::new(file.path(), 1024);
        assert!(writer.is_ok());
        
        let writer = writer.unwrap();
        assert_eq!(writer.capacity(), 1024);
        assert_eq!(writer.position(), 0);
    }
    
    #[test]
    fn test_write() {
        let file = NamedTempFile::new().unwrap();
        let mut writer = MmapWriter::new(file.path(), 1024).unwrap();
        
        let data = b"Hello, World!";
        let written = writer.write(data).unwrap();
        
        assert_eq!(written, data.len());
        assert_eq!(writer.position(), data.len());
    }
    
    #[test]
    fn test_multiple_writes() {
        let file = NamedTempFile::new().unwrap();
        let mut writer = MmapWriter::new(file.path(), 1024).unwrap();
        
        writer.write(b"Hello").unwrap();
        writer.write(b" ").unwrap();
        writer.write(b"World").unwrap();
        
        assert_eq!(writer.position(), 11);
    }
    
    #[test]
    fn test_grow() {
        let file = NamedTempFile::new().unwrap();
        let mut writer = MmapWriter::new(file.path(), 10).unwrap();
        
        let data = b"This is a long string that exceeds initial capacity";
        let result = writer.write(data);
        
        assert!(result.is_ok());
        assert!(writer.capacity() > 10);
    }
    
    #[test]
    fn test_flush() {
        let file = NamedTempFile::new().unwrap();
        let mut writer = MmapWriter::new(file.path(), 1024).unwrap();
        
        writer.write(b"test data").unwrap();
        let result = writer.flush();
        
        assert!(result.is_ok());
    }
}