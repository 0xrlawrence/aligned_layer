use alloy::hex;
use proof_aggregator::aggregators::{risc0_aggregator, sp1_aggregator, zisk_aggregator};
use serde_json::json;
use sp1_sdk::HashableKey;
use std::{fs, path::Path, process::Command};
use tracing::info;
use tracing_subscriber::FmtSubscriber;

const SP1_USER_PROOFS_AGGREGATOR_PROGRAM_ELF: &[u8] =
    include_bytes!("../aggregation_programs/sp1/elf/sp1_user_proofs_aggregator_program");

const SP1_CHUNK_AGGREGATOR_PROGRAM_ELF: &[u8] =
    include_bytes!("../aggregation_programs/sp1/elf/sp1_chunk_aggregator_program");

fn rustc_path_for(toolchain: &str) -> std::path::PathBuf {
    let output = Command::new("rustup")
        .args(["which", "rustc", "--toolchain", toolchain])
        .output()
        .expect("failed to execute rustup");

    if !output.status.success() {
        panic!("rustup which rustc failed for toolchain {toolchain}");
    }

    std::path::PathBuf::from(String::from_utf8_lossy(&output.stdout).trim())
}

fn build_zisk_programs() {
    // Steps followed from https://0xpolygonhermez.github.io/zisk/getting_started/writing_programs.html#build

    // build.rs runs a subprocess without the shell's rustup selection; set the toolchain
    // explicitly so cargo-zisk uses Zisk's rustc instead of the host toolchain.
    let zisk_rustc_path = rustc_path_for("zisk");

    let mut build_command = Command::new("cargo-zisk");

    let mut user_proof_aggregator_rom_vk_command = Command::new("cargo-zisk");
    let mut chunk_aggregator_rom_vk_command = Command::new("cargo-zisk");

    // Zisk build elf command
    build_command
        .env("RUSTC", &zisk_rustc_path)
        .args(["build", "--release"])
        .current_dir("aggregation_programs/zisk/");

    let build_status = build_command
        .status()
        .expect("Failed to execute zisk build command");

    if !build_status.success() {
        panic!("Failed to build zisk elfs");
    }

    // Zisk rom-vkey commands
    let user_proofs_aggregator_rom_vkey_status = user_proof_aggregator_rom_vk_command
        .args([
            "rom-vkey",
            "--elf",
            "./target/riscv64ima-zisk-zkvm-elf/release/zisk_user_proofs_aggregator_program",
            "-o",
            "zisk/vk/zisk_user_proofs_aggregator_program",
        ])
        .env("RUSTC", &zisk_rustc_path)
        .current_dir("./aggregation_programs/")
        .status()
        .unwrap();

    if !user_proofs_aggregator_rom_vkey_status.success() {
        panic!("Failed to execute rom-vkey command on user proofs aggregator program");
    }

    let chunk_aggregator_rom_vkey_status = chunk_aggregator_rom_vk_command
        .args([
            "rom-vkey",
            "--elf",
            "./target/riscv64ima-zisk-zkvm-elf/release/zisk_chunk_aggregator_program",
            "-o",
            "zisk/vk/zisk_chunk_aggregator_program",
        ])
        .env("RUSTC", &zisk_rustc_path)
        .current_dir("./aggregation_programs/")
        .status()
        .unwrap();

    if !chunk_aggregator_rom_vkey_status.success() {
        panic!("Failed to execute rom-vkey command on chunk aggregator program");
    }

    let _ = fs::create_dir("./aggregation_programs/zisk/elf");

    fs::copy(
        "./aggregation_programs/target/riscv64ima-zisk-zkvm-elf/release/zisk_user_proofs_aggregator_program",
        "./aggregation_programs/zisk/elf/zisk_user_proofs_aggregator_program",
    )
    .expect("Could not copy zisk_user_proofs_aggregator_program elf to aggregation_programs/zisk/elf directory");

    fs::copy(
        "./aggregation_programs/target/riscv64ima-zisk-zkvm-elf/release/zisk_chunk_aggregator_program",
        "./aggregation_programs/zisk/elf/zisk_chunk_aggregator_program",
    )
    .expect("Could not copy zisk_chunk_aggregator_program elf to aggregation_programs/zisk/elf directory");
}

fn main() {
    let subscriber = FmtSubscriber::builder().finish();
    tracing::subscriber::set_global_default(subscriber).expect("setting default subscriber failed");

    info!("Building zisk programs...");
    build_zisk_programs();
    info!("Zisk programs built successfully");

    info!(
        "About to write sp1 programs vk hash bytes + risc0 programs image id bytes + zisk rom vk"
    );
    let sp1_user_proofs_aggregator_vk_hash =
        sp1_aggregator::vk_from_elf(SP1_USER_PROOFS_AGGREGATOR_PROGRAM_ELF).bytes32_raw();
    let sp1_user_proofs_aggregator_vk_hash_words =
        sp1_aggregator::vk_from_elf(SP1_USER_PROOFS_AGGREGATOR_PROGRAM_ELF).hash_u32();
    let sp1_chunk_aggregator_vk_hash =
        sp1_aggregator::vk_from_elf(SP1_CHUNK_AGGREGATOR_PROGRAM_ELF).bytes32_raw();

    let risc0_user_proofs_aggregator_image_id_bytes =
        risc0_aggregator::RISC0_USER_PROOFS_AGGREGATOR_PROGRAM_ID_BYTES;
    let risc0_chunk_aggregator_image_id_bytes =
        risc0_aggregator::RISC0_CHUNK_AGGREGATOR_PROGRAM_ID_BYTES;

    let zisk_user_proofs_aggregator_rom_vk = zisk_aggregator::USER_PROOFS_PROGRAM_ROM_VK;
    let zisk_chunk_aggregator_rom_vk_hex = hex::encode(zisk_aggregator::CHUNK_PROGRAM_ROM_VK_BYTES);

    let sp1_user_proofs_aggregator_vk_hash_hex = hex::encode(sp1_user_proofs_aggregator_vk_hash);
    let sp1_chunk_aggregator_vk_hash_hex = hex::encode(sp1_chunk_aggregator_vk_hash);
    let risc0_user_proofs_aggregator_image_id_hex =
        hex::encode(risc0_user_proofs_aggregator_image_id_bytes);
    let risc0_chunk_aggregator_imaged_id_hex = hex::encode(risc0_chunk_aggregator_image_id_bytes);

    let dest_path = Path::new("programs_ids.json");

    let json_data = json!({
        "sp1_user_proofs_aggregator_vk_hash": format!("0x{}", sp1_user_proofs_aggregator_vk_hash_hex),
        "sp1_user_proofs_aggregator_vk_hash_words": format!("{:?}", sp1_user_proofs_aggregator_vk_hash_words),
        "sp1_chunk_aggregator_vk_hash": format!("0x{}", sp1_chunk_aggregator_vk_hash_hex),
        "risc0_user_proofs_aggregator_image_id": format!("0x{}", risc0_user_proofs_aggregator_image_id_hex),
        "risc0_user_proofs_aggregator_image_id_bytes": format!("{:?}", risc0_user_proofs_aggregator_image_id_bytes),
        "risc0_chunk_aggregator_image_id": format!("0x{}", risc0_chunk_aggregator_imaged_id_hex),
        "zisk_user_proofs_aggregator_rom_vk": format!("{:?}", zisk_user_proofs_aggregator_rom_vk),
        "zisk_chunk_aggregator_rom_vk_hex": format!("0x{}", zisk_chunk_aggregator_rom_vk_hex)
    });

    // Write to the file
    fs::write(dest_path, serde_json::to_string_pretty(&json_data).unwrap()).unwrap();

    info!("Program ids written to {:?}", dest_path);
}
