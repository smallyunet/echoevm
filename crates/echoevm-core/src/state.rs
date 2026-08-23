//! EchoEVM-owned account state and transaction journal.

use alloy_primitives::{Address, B256, U256, keccak256};
use std::{
    collections::{BTreeMap, BTreeSet},
    sync::{Arc, Mutex},
};

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct Account {
    pub nonce: u64,
    pub balance: U256,
    pub code: Vec<u8>,
    pub storage: BTreeMap<U256, U256>,
}

impl Account {
    pub fn is_empty(&self) -> bool {
        self.nonce == 0 && self.balance.is_zero() && self.code.is_empty()
    }

    pub fn code_hash(&self) -> B256 {
        if self.code.is_empty() {
            keccak256([])
        } else {
            keccak256(&self.code)
        }
    }
}

#[derive(Clone, Debug, Default)]
pub struct WorldState {
    pub accounts: BTreeMap<Address, Account>,
    pub transient: BTreeMap<(Address, U256), U256>,
    pub warm_addresses: BTreeSet<Address>,
    pub warm_slots: BTreeSet<(Address, U256)>,
    pub original_storage: BTreeMap<(Address, U256), U256>,
    pub created: BTreeSet<Address>,
    pub selfdestructed: BTreeSet<Address>,
    pub refund: i64,
    track_missing: bool,
    known_accounts: BTreeSet<Address>,
    known_slots: BTreeSet<(Address, U256)>,
    missing: Arc<Mutex<MissingReads>>,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct MissingReads {
    pub accounts: BTreeSet<Address>,
    pub storage: BTreeSet<(Address, U256)>,
}

impl WorldState {
    pub fn enable_missing_tracking(&mut self) {
        self.track_missing = true;
    }

    pub fn mark_known_account(&mut self, address: Address) {
        self.known_accounts.insert(address);
    }

    pub fn mark_known_storage(&mut self, address: Address, key: U256) {
        self.known_slots.insert((address, key));
    }

    pub fn missing_reads(&self) -> MissingReads {
        self.missing
            .lock()
            .expect("missing-read tracker lock")
            .clone()
    }

    pub fn account(&self, address: Address) -> Option<&Account> {
        if self.track_missing
            && !self.known_accounts.contains(&address)
            && !self.created.contains(&address)
        {
            self.missing
                .lock()
                .expect("missing-read tracker lock")
                .accounts
                .insert(address);
        }
        self.accounts.get(&address)
    }

    pub fn account_mut(&mut self, address: Address) -> &mut Account {
        if !self.track_missing || self.created.contains(&address) {
            self.known_accounts.insert(address);
        }
        self.accounts.entry(address).or_default()
    }

    pub fn balance(&self, address: Address) -> U256 {
        self.account(address)
            .map(|account| account.balance)
            .unwrap_or_default()
    }

    pub fn code(&self, address: Address) -> &[u8] {
        self.account(address)
            .map(|account| account.code.as_slice())
            .unwrap_or_default()
    }

    pub fn storage(&self, address: Address, key: U256) -> U256 {
        if self.track_missing
            && self.known_accounts.contains(&address)
            && !self.known_slots.contains(&(address, key))
            && !self.created.contains(&address)
        {
            self.missing
                .lock()
                .expect("missing-read tracker lock")
                .storage
                .insert((address, key));
        }
        self.account(address)
            .and_then(|account| account.storage.get(&key).copied())
            .unwrap_or_default()
    }

    pub fn set_storage(&mut self, address: Address, key: U256, value: U256) {
        let account = self.account_mut(address);
        if value.is_zero() {
            account.storage.remove(&key);
        } else {
            account.storage.insert(key, value);
        }
    }

    pub fn transfer(&mut self, from: Address, to: Address, value: U256) -> bool {
        if value.is_zero() {
            return true;
        }
        let balance = self.balance(from);
        if balance < value {
            return false;
        }
        if from == to {
            return true;
        }
        self.account_mut(from).balance = balance - value;
        self.account_mut(to).balance = self.balance(to).wrapping_add(value);
        true
    }

    pub fn begin_transaction(&mut self) {
        self.transient.clear();
        self.warm_addresses.clear();
        self.warm_slots.clear();
        self.original_storage = self
            .accounts
            .iter()
            .flat_map(|(address, account)| {
                account
                    .storage
                    .iter()
                    .map(move |(slot, value)| ((*address, *slot), *value))
            })
            .collect();
        self.created.clear();
        self.selfdestructed.clear();
        self.refund = 0;
    }

    pub fn finalize_transaction(&mut self) {
        for address in std::mem::take(&mut self.selfdestructed) {
            if self.created.contains(&address) {
                self.accounts.remove(&address);
            }
        }
        self.transient.clear();
        self.original_storage.clear();
        self.created.clear();
    }
}
