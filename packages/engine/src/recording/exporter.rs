// packages/engine/src/recording/exporter.rs
//! Export event recordings to various formats
//!
//! Supports:
//! - JSON (for analysis, visualization)
//! - HAR (HTTP Archive format)
//! - JUnit XML (for CI/CD integration)

use crate::recording::recorder::Event;
use crate::utils::errors::{EngineError, Result};
use serde::Serialize;
use tracing::debug;

/// Export formats
#[derive(Debug, Clone, Copy)]
pub enum ExportFormat {
    /// JSON format
    Json,
    
    /// HAR (HTTP Archive) format
    Har,
    
    /// JUnit XML format
    JUnit,
}

/// Exporter for event recordings
pub struct Exporter {
    format: ExportFormat,
}

impl Exporter {
    /// Create a new exporter
    pub fn new(format: ExportFormat) -> Self {
        Self { format }
    }
    
    /// Export events to string
    pub fn export(&self, events: &[Event]) -> Result<String> {
        debug!("Exporting {} events to {:?} format", events.len(), self.format);
        
        match self.format {
            ExportFormat::Json => self.export_json(events),
            ExportFormat::Har => self.export_har(events),
            ExportFormat::JUnit => self.export_junit(events),
        }
    }
    
    /// Export to JSON format
    fn export_json(&self, events: &[Event]) -> Result<String> {
        serde_json::to_string_pretty(events).map_err(|e| {
            EngineError::ExportFailed(format!("JSON serialization error: {}", e))
        })
    }
    
    /// Export to HAR format
    fn export_har(&self, events: &[Event]) -> Result<String> {
        // Filter only HTTP-related events
        let http_events: Vec<_> = events
            .iter()
            .filter(|e| matches!(
                e.event_type,
                crate::recording::recorder::EventType::ExternalCallMade
                    | crate::recording::recorder::EventType::ExternalCallCompleted
            ))
            .collect();
        
        // Build HAR structure
        let har = HarDocument {
            log: HarLog {
                version: "1.2".to_string(),
                creator: HarCreator {
                    name: "Sentra Lab".to_string(),
                    version: env!("CARGO_PKG_VERSION").to_string(),
                },
                entries: http_events
                    .iter()
                    .map(|e| HarEntry {
                        started_date_time: format_timestamp(e.timestamp_ns),
                        time: e.duration_us.unwrap_or(0) as f64 / 1000.0, // Convert to ms
                        request: HarRequest {
                            method: "POST".to_string(), // TODO: Extract from event data
                            url: "http://localhost".to_string(), // TODO: Extract from event data
                        },
                        response: HarResponse {
                            status: 200, // TODO: Extract from event data
                            status_text: "OK".to_string(),
                        },
                    })
                    .collect(),
            },
        };
        
        serde_json::to_string_pretty(&har).map_err(|e| {
            EngineError::ExportFailed(format!("HAR serialization error: {}", e))
        })
    }
    
    /// Export to JUnit XML format
    fn export_junit(&self, events: &[Event]) -> Result<String> {
        // Count test results
        let total = events.len();
        let failures = events
            .iter()
            .filter(|e| {
                matches!(
                    e.event_type,
                    crate::recording::recorder::EventType::ErrorEncountered
                )
            })
            .count();
        
        let xml = format!(
            r#"<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="Sentra Lab Simulation" tests="{}" failures="{}" time="0">
{}</testsuite>"#,
            total,
            failures,
            events
                .iter()
                .map(|e| format!(
                    r#"  <testcase name="{}" time="{}">
{}  </testcase>"#,
                    e.id,
                    e.duration_us.unwrap_or(0) as f64 / 1_000_000.0, // Convert to seconds
                    if matches!(
                        e.event_type,
                        crate::recording::recorder::EventType::ErrorEncountered
                    ) {
                        format!(
                            r#"    <failure message="Error encountered">{}</failure>
"#,
                            serde_json::to_string(&e.data).unwrap_or_default()
                        )
                    } else {
                        String::new()
                    }
                ))
                .collect::<Vec<_>>()
                .join("\n")
        );
        
        Ok(xml)
    }
}

// HAR format structures
#[derive(Serialize)]
struct HarDocument {
    log: HarLog,
}

#[derive(Serialize)]
struct HarLog {
    version: String,
    creator: HarCreator,
    entries: Vec<HarEntry>,
}

#[derive(Serialize)]
struct HarCreator {
    name: String,
    version: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct HarEntry {
    started_date_time: String,
    time: f64,
    request: HarRequest,
    response: HarResponse,
}

#[derive(Serialize)]
struct HarRequest {
    method: String,
    url: String,
}

#[derive(Serialize)]
#[serde(rename_all = "camelCase")]
struct HarResponse {
    status: u16,
    status_text: String,
}

fn format_timestamp(timestamp_ns: u64) -> String {
    use chrono::{DateTime, Utc};
    let secs = (timestamp_ns / 1_000_000_000) as i64;
    let nsecs = (timestamp_ns % 1_000_000_000) as u32;
    let dt = DateTime::<Utc>::from_timestamp(secs, nsecs).unwrap_or_default();
    dt.to_rfc3339()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::recording::recorder::EventType;
    
    fn create_test_event() -> Event {
        Event {
            id: "evt_123".to_string(),
            run_id: "run_abc".to_string(),
            event_type: EventType::AgentStarted,
            timestamp_ns: 1234567890000000000,
            data: serde_json::json!({"test": "data"}),
            duration_us: Some(1000),
        }
    }
    
    #[test]
    fn test_json_export() {
        let exporter = Exporter::new(ExportFormat::Json);
        let events = vec![create_test_event()];
        
        let result = exporter.export(&events);
        assert!(result.is_ok());
        
        let json = result.unwrap();
        assert!(json.contains("evt_123"));
    }
    
    #[test]
    fn test_har_export() {
        let exporter = Exporter::new(ExportFormat::Har);
        let events = vec![create_test_event()];
        
        let result = exporter.export(&events);
        assert!(result.is_ok());
        
        let har = result.unwrap();
        assert!(har.contains("Sentra Lab"));
        assert!(har.contains("version"));
    }
    
    #[test]
    fn test_junit_export() {
        let exporter = Exporter::new(ExportFormat::JUnit);
        let events = vec![create_test_event()];
        
        let result = exporter.export(&events);
        assert!(result.is_ok());
        
        let xml = result.unwrap();
        assert!(xml.contains("<?xml"));
        assert!(xml.contains("testsuite"));
        assert!(xml.contains("evt_123"));
    }
}