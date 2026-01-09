pub const USER_PROOFS_PROGRAM_ROM_VK_BYTES: &[u8] =
    include_bytes!("../../aggregation_programs/zisk/vk/zisk_user_proofs_aggregator_program");

pub const CHUNK_PROGRAM_ROM_VK_BYTES: &[u8] =
    include_bytes!("../../aggregation_programs/zisk/vk/zisk_chunk_aggregator_program");

pub const USER_PROOFS_PROGRAM_ROM_VK: [u64; 4] =
    vk_bytes_to_u64_4(USER_PROOFS_PROGRAM_ROM_VK_BYTES);
pub const CHUNK_PROGRAM_ROM_VK: [u64; 4] = vk_bytes_to_u64_4(CHUNK_PROGRAM_ROM_VK_BYTES);

const fn vk_bytes_to_u64_4(bytes: &[u8]) -> [u64; 4] {
    let mut out = [0_u64; 4];
    let mut i = 0;
    while i < 4 {
        let base = i * 8;
        out[i] = u64::from_le_bytes([
            bytes[base],
            bytes[base + 1],
            bytes[base + 2],
            bytes[base + 3],
            bytes[base + 4],
            bytes[base + 5],
            bytes[base + 6],
            bytes[base + 7],
        ]);
        i += 1;
    }
    out
}
