// packages/engine/build.rs
//! Build script for compiling Protocol Buffer definitions

use std::io::Result;

fn main() -> Result<()> {
    // Compile protobuf files
    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .out_dir("src/generated")
        .compile(
            &[
                "proto/engine.proto",
                "proto/events.proto",
                "proto/state.proto",
            ],
            &["proto"],
        )?;

    // Recompile if proto files change
    println!("cargo:rerun-if-changed=proto/engine.proto");
    println!("cargo:rerun-if-changed=proto/events.proto");
    println!("cargo:rerun-if-changed=proto/state.proto");

    Ok(())
}