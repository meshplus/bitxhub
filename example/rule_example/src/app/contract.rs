// extern crate protobuf;
// extern crate sha2;

// use crate::crypto::ecdsa;
// use crate::model::transaction;
// use sha2::{Digest, Sha256};

pub fn verify(proof: &[u8], validator: &[u8]) -> bool {
  // let cap =
  //   protobuf::parse_from_bytes::<transaction::ChaincodeActionPayload>(proof).expect("error");
  // let cap_act = cap.action.unwrap();
  // let prp = protobuf::parse_from_bytes::<transaction::ProposalResponsePayload>(
  //   &cap_act.proposal_response_payload,
  // )
  // .expect("error");
  // println!("{:?}", prp);

  // let endorsers = cap_act.endorsements;
  // println!("{:?}", endorsers[0]);

  // let mut digest = Sha256::new();
  // let mut payload = cap_act.proposal_response_payload.to_owned();
  // payload.extend(&endorsers[0].endorser);
  // digest.input(&payload);
  // let digest_byte = digest.result();
  // println!("{:?}", digest_byte);

  // return ecdsa::verify(
  //   &endorsers[0].signature,
  //   &digest_byte,
  //   &validator,
  //   ecdsa::EcdsaAlgorithmn::P256,
  // );
  return true
}
