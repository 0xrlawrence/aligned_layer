#![no_main]
ziskos::entrypoint!(main);

use lambdaworks_crypto::merkle_tree::merkle::MerkleTree;
use zisk_aggregation_program::{ChunkAggregatorInput, Hash32};

// Generated with `make proof_aggregator_write_program_ids` and copied from program_ids.json
pub const USER_PROOFS_AGGREGATOR_PROGRAM_VK_HASH: [u8; 32] = [0u8; 32];

pub fn main() {
    let input = ziskos::read_input_slice();
    let input = bincode::deserialize::<ChunkAggregatorInput>(&input).unwrap();

    let mut leaves = vec![];

    // Verify the proofs.
    for (proof, leaves_commitment) in input.proofs_and_leaves_commitment {
        // Ensure the aggregated chunk originates from the user proofs aggregation program.
        // This validation step guarantees that the proof was genuinely verified
        // by this program. Without this check, a different program using the
        // same public inputs could bypass verification.
        assert!(proof.vk.clone() == USER_PROOFS_AGGREGATOR_PROGRAM_VK_HASH);

        let merkle_root: [u8; 32] = proof
            .proof
            .clone()
            .try_into()
            .expect("Public input to be the hash of the chunk tree");

        // Reconstruct the merkle tree and verify that the roots match
        let leaves_commitment: Vec<Hash32> = leaves_commitment.into_iter().map(Hash32).collect();
        let merkle_tree: MerkleTree<Hash32> = MerkleTree::build(&leaves_commitment).unwrap();
        assert!(merkle_tree.root == merkle_root);

        leaves.extend(leaves_commitment);

        proofman_verifier::verify(&proof.proof, &proof.vk);
    }

    // Finally, compute the final merkle root with all the leaves
    let merkle_tree: MerkleTree<Hash32> = MerkleTree::build(&leaves).unwrap();

    merkle_tree
        .root
        .chunks_exact(4)
        .enumerate()
        .for_each(|(idx, bytes)| {
            ziskos::set_output(idx, u32::from_le_bytes(bytes.try_into().unwrap()))
        });
}
