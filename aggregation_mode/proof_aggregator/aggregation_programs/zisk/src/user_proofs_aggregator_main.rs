#![no_main]
ziskos::entrypoint!(main);

use lambdaworks_crypto::merkle_tree::merkle::MerkleTree;
use zisk_aggregation_program::{UserProofsAggregatorInput, ZiskProofAndVk};

fn main() {
    let input = ziskos::read_input_slice();
    let input =
        bincode::deserialize::<UserProofsAggregatorInput>(&input).expect("correct serialization");

    for entry in input.proofs_and_vk.iter() {
        proofman_verifier::verify(&entry.proof, &entry.vk);
    }

    let merkle_tree = MerkleTree::<ZiskProofAndVk>::build(&input.proofs_and_vk).unwrap();

    merkle_tree
        .root
        .chunks_exact(4)
        .enumerate()
        .for_each(|(idx, bytes)| {
            ziskos::set_output(idx, u32::from_le_bytes(bytes.try_into().unwrap()))
        });
}
