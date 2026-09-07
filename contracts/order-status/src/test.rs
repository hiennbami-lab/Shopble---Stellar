#![cfg(test)]

use super::*;
use soroban_sdk::{testutils::Address as _, Env};

fn setup() -> (Env, OrderStatusContractClient<'static>, Address, Address) {
    let env = Env::default();
    env.mock_all_auths();
    let operator = Address::generate(&env);
    let buyer = Address::generate(&env);
    let id = env.register(OrderStatusContract, (operator.clone(),));
    let client = OrderStatusContractClient::new(&env, &id);
    (env, client, operator, buyer)
}

fn seed(env: &Env, client: &OrderStatusContractClient, buyer: &Address, id: &str) -> String {
    let order_id = String::from_str(env, id);
    client.create_order(
        &order_id,
        buyer,
        &25_5000000i128,
        &String::from_str(env, "USDC"),
        &BytesN::from_array(env, &[7u8; 32]),
    );
    order_id
}

#[test]
fn create_stores_all_fields_in_opening_state() {
    let (env, client, _op, buyer) = setup();
    let id = seed(&env, &client, &buyer, "ord1");

    let o = client.get_order(&id);
    assert_eq!(o.status, Status::AwaitingPayment);
    assert_eq!(o.buyer, buyer);
    assert_eq!(o.expected_amount, 25_5000000i128);
    assert_eq!(o.asset, String::from_str(&env, "USDC"));
    assert_eq!(o.memo_hash, BytesN::from_array(&env, &[7u8; 32]));
    assert!(!client.is_fulfillable(&id));
}

#[test]
fn happy_path_reaches_validated_and_is_fulfillable() {
    let (env, client, _op, buyer) = setup();
    let id = seed(&env, &client, &buyer, "ord1");

    client.set_status(&id, &Status::PaymentDetected);
    assert_eq!(client.get_order(&id).status, Status::PaymentDetected);
    assert!(!client.is_fulfillable(&id));

    client.set_status(&id, &Status::Validated);
    assert_eq!(client.get_order(&id).status, Status::Validated);
    assert!(client.is_fulfillable(&id));
}

#[test]
fn rejection_carries_its_named_reason() {
    let (env, client, _op, buyer) = setup();
    let id = seed(&env, &client, &buyer, "ord1");

    client.set_status(&id, &Status::PaymentDetected);
    client.set_status(&id, &Status::Rejected(RejectReason::Underpayment));

    assert_eq!(
        client.get_order(&id).status,
        Status::Rejected(RejectReason::Underpayment)
    );
    // Rejected is never fulfilment-eligible.
    assert!(!client.is_fulfillable(&id));
}

#[test]
fn every_reject_reason_round_trips() {
    let (env, client, _op, buyer) = setup();
    let reasons = [
        RejectReason::Underpayment,
        RejectReason::WrongAsset,
        RejectReason::WrongDestination,
        RejectReason::WrongBuyerWallet,
        RejectReason::InvalidMemo,
    ];
    for (i, r) in reasons.iter().enumerate() {
        let id = seed(&env, &client, &buyer, match i {
            0 => "r0", 1 => "r1", 2 => "r2", 3 => "r3", _ => "r4",
        });
        client.set_status(&id, &Status::PaymentDetected);
        client.set_status(&id, &Status::Rejected(r.clone()));
        assert_eq!(client.get_order(&id).status, Status::Rejected(r.clone()));
    }
}

#[test]
fn cannot_skip_payment_detected() {
    let (env, client, _op, buyer) = setup();
    let id = seed(&env, &client, &buyer, "ord1");

    // This is the guard that matters: a backend bug must not be able to mark an
    // order fulfilment-eligible without a payment having been observed.
    assert_eq!(
        client.try_set_status(&id, &Status::Validated),
        Err(Ok(Error::InvalidTransition))
    );
    assert_eq!(client.get_order(&id).status, Status::AwaitingPayment);
}

#[test]
fn settled_orders_are_final() {
    let (env, client, _op, buyer) = setup();
    let id = seed(&env, &client, &buyer, "ord1");
    client.set_status(&id, &Status::PaymentDetected);
    client.set_status(&id, &Status::Validated);

    // A duplicate payment arriving later must not rewrite the verdict.
    assert_eq!(
        client.try_set_status(&id, &Status::Rejected(RejectReason::Underpayment)),
        Err(Ok(Error::InvalidTransition))
    );
    assert_eq!(client.get_order(&id).status, Status::Validated);
}

#[test]
fn duplicate_create_is_rejected() {
    let (env, client, _op, buyer) = setup();
    let id = seed(&env, &client, &buyer, "ord1");

    let res = client.try_create_order(
        &id,
        &buyer,
        &1i128,
        &String::from_str(&env, "USDC"),
        &BytesN::from_array(&env, &[0u8; 32]),
    );
    assert_eq!(res, Err(Ok(Error::AlreadyExists)));
    // Original is untouched.
    assert_eq!(client.get_order(&id).expected_amount, 25_5000000i128);
}

#[test]
fn unknown_order_is_not_found() {
    let (env, client, _op, _buyer) = setup();
    let missing = String::from_str(&env, "nope");

    assert_eq!(client.try_get_order(&missing), Err(Ok(Error::NotFound)));
    assert_eq!(
        client.try_set_status(&missing, &Status::PaymentDetected),
        Err(Ok(Error::NotFound))
    );
    assert!(!client.is_fulfillable(&missing));
}

#[test]
fn operator_is_fixed_at_deploy() {
    let (_env, client, op, _buyer) = setup();
    assert_eq!(client.get_operator(), op);
}

#[test]
#[should_panic]
fn writes_require_operator_auth() {
    // No mock_all_auths here: the require_auth must actually bite.
    let env = Env::default();
    let operator = Address::generate(&env);
    let buyer = Address::generate(&env);
    let id = env.register(OrderStatusContract, (operator,));
    let client = OrderStatusContractClient::new(&env, &id);

    client.create_order(
        &String::from_str(&env, "ord1"),
        &buyer,
        &1i128,
        &String::from_str(&env, "USDC"),
        &BytesN::from_array(&env, &[0u8; 32]),
    );
}
