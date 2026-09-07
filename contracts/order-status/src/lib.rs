#![no_std]

//! Shopble order-status registry (SOW Deliverable 3).
//!
//! Deliberately minimal: it stores order status and nothing else. It holds no
//! tokens, performs no transfers, implements no escrow and makes no
//! cross-contract calls.
//!
//! The point of putting status on-chain is that a merchant or third party can
//! check whether an order is fulfilment-eligible WITHOUT trusting Shopble's
//! backend. So the state machine is enforced here too, not just in Postgres —
//! a backend bug cannot drive an order straight to `Validated` without the
//! `PaymentDetected` step that says a payment was actually observed.

use soroban_sdk::{
    contract, contracterror, contractimpl, contracttype, Address, BytesN, Env, String,
};

#[contracttype]
#[derive(Clone)]
pub enum DataKey {
    Operator,
    Order(String),
}

/// The five named rejection reasons from the SOW. A closed enum: the contract
/// cannot record a rejection that has no name, which is what makes a rejected
/// order auditable rather than merely plausible.
#[contracttype]
#[derive(Clone, PartialEq, Eq, Debug)]
pub enum RejectReason {
    Underpayment,
    WrongAsset,
    WrongDestination,
    WrongBuyerWallet,
    InvalidMemo,
}

/// awaiting_payment -> payment_detected -> validated | rejected(reason).
/// Only `Validated` marks an order fulfilment-eligible.
#[contracttype]
#[derive(Clone, PartialEq, Eq, Debug)]
pub enum Status {
    AwaitingPayment,
    PaymentDetected,
    Validated,
    Rejected(RejectReason),
}

#[contracttype]
#[derive(Clone, PartialEq, Eq, Debug)]
pub struct Order {
    pub buyer: Address,
    pub expected_amount: i128,
    pub asset: String,
    pub memo_hash: BytesN<32>,
    pub status: Status,
}

#[contracterror]
#[derive(Copy, Clone, PartialEq, Eq, Debug, PartialOrd, Ord)]
#[repr(u32)]
pub enum Error {
    AlreadyExists = 1,
    NotFound = 2,
    InvalidTransition = 3,
}

/// The only legal edges. Everything else is rejected, including any transition
/// out of a final state — a payment arriving after an order is settled must not
/// be able to rewrite the verdict.
fn can_transition(from: &Status, to: &Status) -> bool {
    matches!(
        (from, to),
        (&Status::AwaitingPayment, &Status::PaymentDetected)
            | (&Status::PaymentDetected, &Status::Validated)
            | (&Status::PaymentDetected, &Status::Rejected(_))
    )
}

fn operator(env: &Env) -> Address {
    env.storage()
        .instance()
        .get(&DataKey::Operator)
        .expect("contract not initialised")
}

#[contract]
pub struct OrderStatusContract;

#[contractimpl]
impl OrderStatusContract {
    /// Operator is fixed at deploy time. Using a constructor rather than an
    /// `initialize` entrypoint removes the window where anyone could claim an
    /// uninitialised contract by calling it first.
    pub fn __constructor(env: Env, operator: Address) {
        env.storage().instance().set(&DataKey::Operator, &operator);
    }

    /// Register an order intent on-chain in its opening state.
    ///
    /// `memo_hash` is a hash, not the memo itself: the memo is the order id and
    /// putting it in clear would publish the reference that links a buyer wallet
    /// to a specific purchase. A hash still lets anyone verify a memo they
    /// already know.
    pub fn create_order(
        env: Env,
        order_id: String,
        buyer: Address,
        expected_amount: i128,
        asset: String,
        memo_hash: BytesN<32>,
    ) -> Result<(), Error> {
        operator(&env).require_auth();

        let key = DataKey::Order(order_id);
        if env.storage().persistent().has(&key) {
            return Err(Error::AlreadyExists);
        }
        env.storage().persistent().set(
            &key,
            &Order {
                buyer,
                expected_amount,
                asset,
                memo_hash,
                status: Status::AwaitingPayment,
            },
        );
        Ok(())
    }

    /// Record a verdict. Operator-guarded and transition-guarded.
    pub fn set_status(env: Env, order_id: String, status: Status) -> Result<(), Error> {
        operator(&env).require_auth();

        let key = DataKey::Order(order_id);
        let mut order: Order = env
            .storage()
            .persistent()
            .get(&key)
            .ok_or(Error::NotFound)?;

        if !can_transition(&order.status, &status) {
            return Err(Error::InvalidTransition);
        }
        order.status = status;
        env.storage().persistent().set(&key, &order);
        Ok(())
    }

    /// Read an order. Unauthenticated on purpose — the whole point is that a
    /// third party can check status without trusting anyone.
    pub fn get_order(env: Env, order_id: String) -> Result<Order, Error> {
        env.storage()
            .persistent()
            .get(&DataKey::Order(order_id))
            .ok_or(Error::NotFound)
    }

    /// Fulfilment gate, on-chain. One call, so a merchant does not have to know
    /// how the status enum is encoded to answer the only question that matters.
    pub fn is_fulfillable(env: Env, order_id: String) -> bool {
        match env
            .storage()
            .persistent()
            .get::<DataKey, Order>(&DataKey::Order(order_id))
        {
            Some(o) => o.status == Status::Validated,
            None => false,
        }
    }

    pub fn get_operator(env: Env) -> Address {
        operator(&env)
    }
}

mod test;
