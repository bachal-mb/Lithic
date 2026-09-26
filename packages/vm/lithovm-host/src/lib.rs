use lithovm::{
    EventRecord, ExecutionContext, ExecutionFailure, ExecutionOutcome, ExecutionResult,
    FailureKind, Storage, Vm,
};
use lithovm_bytecode::{function_selector, parse};
use sha3::{Digest, Keccak256};
use std::collections::BTreeMap;

pub type Address = [u8; 32];
pub type Selector = [u8; 32];

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeployedContract {
    pub bytecode: Vec<u8>,
    pub code_hash: [u8; 32],
    pub storage: Storage,
    pub entrypoints: BTreeMap<Selector, String>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Initializer {
    pub function: String,
    pub arguments: Vec<[u8; 32]>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeployRequest {
    pub deployer: Address,
    pub nonce: u64,
    pub bytecode: Vec<u8>,
    pub initializer: Option<Initializer>,
    pub value: [u8; 32],
    pub gas_limit: u64,
    pub block_height: u64,
    pub block_timestamp: u64,
    pub chain_id: u64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct DeploySuccess {
    pub contract: Address,
    pub code_hash: [u8; 32],
    pub initializer_result: Option<ExecutionResult>,
    pub gas_used: u64,
    pub events: Vec<CommittedEvent>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum DeployOutcome {
    Success(DeploySuccess),
    Failure(HostFailure),
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CallRequest {
    pub contract: Address,
    pub function: String,
    pub arguments: Vec<[u8; 32]>,
    pub gas_limit: u64,
    pub caller: Address,
    pub value: [u8; 32],
    pub block_height: u64,
    pub block_timestamp: u64,
    pub chain_id: u64,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct CommittedEvent {
    pub contract: Address,
    pub event: EventRecord,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct HostSuccess {
    pub result: ExecutionResult,
    pub gas_used: u64,
    pub events: Vec<CommittedEvent>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum HostFailureKind {
    Vm(FailureKind),
    MissingContract,
    UnknownSelector,
    Reentrancy,
    InvalidBytecode,
    InvalidEntrypoint,
    ContractExists,
    InsufficientBalance,
    State,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct HostFailure {
    pub kind: HostFailureKind,
    pub message: String,
    pub gas_used: u64,
    pub failed_contract: Address,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum HostOutcome {
    Success(HostSuccess),
    Failure(HostFailure),
}

/// A transaction-scoped view of chain state. Implementations must make
/// `commit` atomic and discard every write when dropped without committing.
pub trait StateTransaction {
    fn load_contract(&self, address: &Address) -> Result<Option<DeployedContract>, String>;
    fn store_contract(
        &mut self,
        address: Address,
        contract: DeployedContract,
    ) -> Result<(), String>;
    fn load_balance(&self, address: &Address) -> Result<[u8; 32], String>;
    fn store_balance(&mut self, address: Address, balance: [u8; 32]) -> Result<(), String>;
    fn commit(self) -> Result<(), String>;
}

/// Persistence seam implemented by the chain and by the conformance adapter.
pub trait TransactionalState {
    type Transaction<'a>: StateTransaction
    where
        Self: 'a;

    fn begin_transaction(&mut self) -> Result<Self::Transaction<'_>, String>;
}

pub struct TransactionalHost<S> {
    vm: Vm,
    state: S,
}

impl<S> TransactionalHost<S> {
    pub fn new(state: S) -> Self {
        Self {
            vm: Vm::default(),
            state,
        }
    }

    pub fn state(&self) -> &S {
        &self.state
    }

    pub fn state_mut(&mut self) -> &mut S {
        &mut self.state
    }

    pub fn into_state(self) -> S {
        self.state
    }
}

impl<S: TransactionalState> TransactionalHost<S> {
    pub fn deploy(&mut self, request: DeployRequest) -> DeployOutcome {
        let code_hash = code_hash(&request.bytecode);
        let address =
            contract_address(request.deployer, request.nonce, code_hash, request.chain_id);
        let program = match parse(&request.bytecode) {
            Ok(program) => program,
            Err(error) => {
                return DeployOutcome::Failure(HostFailure {
                    kind: HostFailureKind::InvalidBytecode,
                    message: error.to_string(),
                    gas_used: 0,
                    failed_contract: address,
                })
            }
        };
        let mut entrypoints = BTreeMap::new();
        for function in &program.functions {
            if entrypoints
                .insert(function_selector(function), function.name.clone())
                .is_some()
            {
                return DeployOutcome::Failure(HostFailure {
                    kind: HostFailureKind::InvalidEntrypoint,
                    message: "canonical function selector collision".into(),
                    gas_used: 0,
                    failed_contract: address,
                });
            }
        }
        let mut transaction = match self.state.begin_transaction() {
            Ok(transaction) => transaction,
            Err(message) => return DeployOutcome::Failure(state_failure(message, 0, address)),
        };
        match transaction.load_contract(&address) {
            Ok(Some(_)) => {
                return DeployOutcome::Failure(HostFailure {
                    kind: HostFailureKind::ContractExists,
                    message: "derived contract address is already deployed".into(),
                    gas_used: 0,
                    failed_contract: address,
                })
            }
            Ok(None) => {}
            Err(message) => {
                return DeployOutcome::Failure(state_failure(message, 0, address));
            }
        }
        let deployer_balance = match load_balance(&transaction, request.deployer, 0, address) {
            Ok(balance) => balance,
            Err(failure) => return DeployOutcome::Failure(failure),
        };
        let Some(deployer_balance) = debit(deployer_balance, request.value) else {
            return DeployOutcome::Failure(HostFailure {
                kind: HostFailureKind::InsufficientBalance,
                message: "deployment value exceeds deployer balance".into(),
                gas_used: 0,
                failed_contract: address,
            });
        };
        if let Err(failure) = store_balance(
            &mut transaction,
            request.deployer,
            deployer_balance,
            0,
            address,
        ) {
            return DeployOutcome::Failure(failure);
        }
        if let Err(failure) = store_balance(&mut transaction, address, request.value, 0, address) {
            return DeployOutcome::Failure(failure);
        }
        if let Err(failure) = store_contract(
            &mut transaction,
            address,
            DeployedContract {
                bytecode: request.bytecode,
                code_hash,
                storage: Storage::default(),
                entrypoints,
            },
            0,
        ) {
            return DeployOutcome::Failure(failure);
        }

        let mut events = Vec::new();
        let mut active_contracts = vec![address];
        let initialized = if let Some(initializer) = request.initializer {
            let frame = FrameRequest {
                contract: address,
                function: initializer.function,
                arguments: initializer.arguments,
                gas_limit: request.gas_limit,
                caller: request.deployer,
                value: request.value,
                block_height: request.block_height,
                block_timestamp: request.block_timestamp,
                chain_id: request.chain_id,
                depth: 0,
            };
            match execute_frame(
                &self.vm,
                &mut transaction,
                frame,
                &mut events,
                &mut active_contracts,
            ) {
                Ok((result, gas_used)) => Some((result, gas_used)),
                Err(failure) => return DeployOutcome::Failure(failure),
            }
        } else {
            None
        };
        let (initializer_result, gas_used) = initialized
            .map(|(result, gas)| (Some(result), gas))
            .unwrap_or((None, 0));
        match transaction.commit() {
            Ok(()) => DeployOutcome::Success(DeploySuccess {
                contract: address,
                code_hash,
                initializer_result,
                gas_used,
                events,
            }),
            Err(message) => DeployOutcome::Failure(state_failure(message, gas_used, address)),
        }
    }

    pub fn execute(&mut self, request: CallRequest) -> HostOutcome {
        let mut transaction = match self.state.begin_transaction() {
            Ok(transaction) => transaction,
            Err(message) => {
                return HostOutcome::Failure(HostFailure {
                    kind: HostFailureKind::State,
                    message,
                    gas_used: 0,
                    failed_contract: request.contract,
                })
            }
        };
        let mut events = Vec::new();
        let mut active_contracts = vec![request.contract];
        let frame = FrameRequest {
            contract: request.contract,
            function: request.function,
            arguments: request.arguments,
            gas_limit: request.gas_limit,
            caller: request.caller,
            value: request.value,
            block_height: request.block_height,
            block_timestamp: request.block_timestamp,
            chain_id: request.chain_id,
            depth: 0,
        };
        match execute_frame(
            &self.vm,
            &mut transaction,
            frame,
            &mut events,
            &mut active_contracts,
        ) {
            Ok((result, gas_used)) => match transaction.commit() {
                Ok(()) => HostOutcome::Success(HostSuccess {
                    result,
                    gas_used,
                    events,
                }),
                Err(message) => HostOutcome::Failure(HostFailure {
                    kind: HostFailureKind::State,
                    message,
                    gas_used,
                    failed_contract: request.contract,
                }),
            },
            Err(failure) => HostOutcome::Failure(failure),
        }
    }
}

pub fn code_hash(bytecode: &[u8]) -> [u8; 32] {
    Keccak256::digest(bytecode).into()
}

pub fn contract_address(
    deployer: Address,
    nonce: u64,
    code_hash: [u8; 32],
    chain_id: u64,
) -> Address {
    let mut hasher = Keccak256::new();
    hasher.update(b"LITHOVM_DEPLOY_V1");
    hasher.update(chain_id.to_be_bytes());
    hasher.update(deployer);
    hasher.update(nonce.to_be_bytes());
    hasher.update(code_hash);
    let digest = hasher.finalize();
    let mut address = [0; 32];
    address[12..].copy_from_slice(&digest[12..]);
    address
}

struct FrameRequest {
    contract: Address,
    function: String,
    arguments: Vec<[u8; 32]>,
    gas_limit: u64,
    caller: Address,
    value: [u8; 32],
    block_height: u64,
    block_timestamp: u64,
    chain_id: u64,
    depth: u16,
}

fn execute_frame<T: StateTransaction>(
    vm: &Vm,
    state: &mut T,
    frame: FrameRequest,
    committed_events: &mut Vec<CommittedEvent>,
    active_contracts: &mut Vec<Address>,
) -> Result<(ExecutionResult, u64), HostFailure> {
    let mut contract = load_contract(state, frame.contract, 0)?;
    let contract_balance = load_balance(state, frame.contract, 0, frame.contract)?;
    let context = ExecutionContext {
        caller: frame.caller,
        value: frame.value,
        block_height: frame.block_height,
        block_timestamp: frame.block_timestamp,
        chain_id: frame.chain_id,
        contract_balance,
        call_depth: frame.depth,
    };
    let result = match vm.execute_transactionally(
        &contract.bytecode,
        &frame.function,
        &frame.arguments,
        frame.gas_limit,
        &mut contract.storage,
        &context,
    ) {
        ExecutionOutcome::Success(result) => result,
        ExecutionOutcome::Failure(failure) => {
            return Err(vm_failure(frame.contract, failure));
        }
    };
    let mut total_gas = result.gas_used;

    // LithoVM calls are deferred effects. Persist the caller's storage before
    // entering children so reentrant frames observe checks/effects updates.
    store_contract(state, frame.contract, contract, total_gas)?;
    committed_events.extend(result.events.iter().cloned().map(|event| CommittedEvent {
        contract: frame.contract,
        event,
    }));

    for transfer in &result.transfers {
        let contract_balance = load_balance(state, frame.contract, total_gas, frame.contract)?;
        let contract_balance =
            debit(contract_balance, transfer.amount).ok_or_else(|| HostFailure {
                kind: HostFailureKind::State,
                message: "VM emitted a transfer exceeding the persisted contract balance".into(),
                gas_used: total_gas,
                failed_contract: frame.contract,
            })?;
        store_balance(
            state,
            frame.contract,
            contract_balance,
            total_gas,
            frame.contract,
        )?;
        let recipient_balance = load_balance(state, transfer.recipient, total_gas, frame.contract)?;
        let recipient_balance =
            credit(recipient_balance, transfer.amount).ok_or_else(|| HostFailure {
                kind: HostFailureKind::State,
                message: "native recipient balance overflow".into(),
                gas_used: total_gas,
                failed_contract: frame.contract,
            })?;
        store_balance(
            state,
            transfer.recipient,
            recipient_balance,
            total_gas,
            frame.contract,
        )?;
    }

    for call in &result.calls {
        if active_contracts.contains(&call.target) {
            return Err(HostFailure {
                kind: HostFailureKind::Reentrancy,
                message: "reentrant contract call is not permitted".into(),
                gas_used: total_gas,
                failed_contract: call.target,
            });
        }
        let contract_balance = load_balance(state, frame.contract, total_gas, frame.contract)?;
        let contract_balance = debit(contract_balance, call.value).ok_or_else(|| HostFailure {
            kind: HostFailureKind::State,
            message: "VM emitted a call exceeding the persisted contract balance".into(),
            gas_used: total_gas,
            failed_contract: frame.contract,
        })?;
        store_balance(
            state,
            frame.contract,
            contract_balance,
            total_gas,
            frame.contract,
        )?;
        let child = load_contract(state, call.target, total_gas)?;
        let function = child
            .entrypoints
            .get(&call.selector)
            .cloned()
            .ok_or_else(|| HostFailure {
                kind: HostFailureKind::UnknownSelector,
                message: "contract selector is not registered".into(),
                gas_used: total_gas,
                failed_contract: call.target,
            })?;
        let child_balance = load_balance(state, call.target, total_gas, call.target)?;
        let child_balance = credit(child_balance, call.value).ok_or_else(|| HostFailure {
            kind: HostFailureKind::State,
            message: "callee balance overflow".into(),
            gas_used: total_gas,
            failed_contract: call.target,
        })?;
        store_balance(state, call.target, child_balance, total_gas, call.target)?;
        let remaining_gas = frame.gas_limit.saturating_sub(total_gas);
        let child_frame = FrameRequest {
            contract: call.target,
            function,
            arguments: Vec::new(),
            gas_limit: remaining_gas,
            caller: frame.contract,
            value: call.value,
            block_height: frame.block_height,
            block_timestamp: frame.block_timestamp,
            chain_id: frame.chain_id,
            depth: call.depth,
        };
        active_contracts.push(call.target);
        let child_result =
            execute_frame(vm, state, child_frame, committed_events, active_contracts);
        active_contracts.pop();
        match child_result {
            Ok((_, child_gas)) => total_gas = total_gas.saturating_add(child_gas),
            Err(mut failure) => {
                failure.gas_used = total_gas
                    .saturating_add(failure.gas_used)
                    .min(frame.gas_limit);
                return Err(failure);
            }
        }
    }

    Ok((result, total_gas))
}

fn vm_failure(contract: Address, failure: ExecutionFailure) -> HostFailure {
    HostFailure {
        kind: HostFailureKind::Vm(failure.kind),
        message: failure.message,
        gas_used: failure.gas_used,
        failed_contract: contract,
    }
}

fn load_contract<T: StateTransaction>(
    state: &T,
    address: Address,
    gas_used: u64,
) -> Result<DeployedContract, HostFailure> {
    let contract = state
        .load_contract(&address)
        .map_err(|message| state_failure(message, gas_used, address))?
        .ok_or_else(|| HostFailure {
            kind: HostFailureKind::MissingContract,
            message: "contract is not deployed".into(),
            gas_used,
            failed_contract: address,
        })?;
    if code_hash(&contract.bytecode) != contract.code_hash {
        return Err(HostFailure {
            kind: HostFailureKind::State,
            message: "persisted contract code hash does not match bytecode".into(),
            gas_used,
            failed_contract: address,
        });
    }
    Ok(contract)
}

fn store_contract<T: StateTransaction>(
    state: &mut T,
    address: Address,
    contract: DeployedContract,
    gas_used: u64,
) -> Result<(), HostFailure> {
    state
        .store_contract(address, contract)
        .map_err(|message| state_failure(message, gas_used, address))
}

fn load_balance<T: StateTransaction>(
    state: &T,
    address: Address,
    gas_used: u64,
    contract: Address,
) -> Result<[u8; 32], HostFailure> {
    state
        .load_balance(&address)
        .map_err(|message| state_failure(message, gas_used, contract))
}

fn store_balance<T: StateTransaction>(
    state: &mut T,
    address: Address,
    balance: [u8; 32],
    gas_used: u64,
    contract: Address,
) -> Result<(), HostFailure> {
    state
        .store_balance(address, balance)
        .map_err(|message| state_failure(message, gas_used, contract))
}

fn state_failure(message: String, gas_used: u64, contract: Address) -> HostFailure {
    HostFailure {
        kind: HostFailureKind::State,
        message,
        gas_used,
        failed_contract: contract,
    }
}

fn debit(left: [u8; 32], right: [u8; 32]) -> Option<[u8; 32]> {
    if left < right {
        return None;
    }
    let mut result = [0; 32];
    let mut borrow = 0i16;
    for index in (0..32).rev() {
        let value = left[index] as i16 - right[index] as i16 - borrow;
        if value < 0 {
            result[index] = (value + 256) as u8;
            borrow = 1;
        } else {
            result[index] = value as u8;
            borrow = 0;
        }
    }
    Some(result)
}

fn credit(left: [u8; 32], right: [u8; 32]) -> Option<[u8; 32]> {
    let mut result = [0; 32];
    let mut carry = 0u16;
    for index in (0..32).rev() {
        let value = left[index] as u16 + right[index] as u16 + carry;
        result[index] = value as u8;
        carry = value >> 8;
    }
    (carry == 0).then_some(result)
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct InMemoryState {
    contracts: BTreeMap<Address, DeployedContract>,
    balances: BTreeMap<Address, [u8; 32]>,
}

impl InMemoryState {
    pub fn deploy(&mut self, address: Address, contract: DeployedContract) {
        self.contracts.insert(address, contract);
    }

    pub fn contract(&self, address: &Address) -> Option<&DeployedContract> {
        self.contracts.get(address)
    }

    pub fn balance(&self, address: &Address) -> [u8; 32] {
        self.balances.get(address).copied().unwrap_or([0; 32])
    }

    pub fn set_balance(&mut self, address: Address, balance: [u8; 32]) {
        self.balances.insert(address, balance);
    }
}

pub struct InMemoryTransaction<'a> {
    parent: &'a mut InMemoryState,
    staged: InMemoryState,
}

impl TransactionalState for InMemoryState {
    type Transaction<'a> = InMemoryTransaction<'a>;

    fn begin_transaction(&mut self) -> Result<Self::Transaction<'_>, String> {
        let staged = self.clone();
        Ok(InMemoryTransaction {
            parent: self,
            staged,
        })
    }
}

impl StateTransaction for InMemoryTransaction<'_> {
    fn load_contract(&self, address: &Address) -> Result<Option<DeployedContract>, String> {
        Ok(self.staged.contracts.get(address).cloned())
    }

    fn store_contract(
        &mut self,
        address: Address,
        contract: DeployedContract,
    ) -> Result<(), String> {
        self.staged.contracts.insert(address, contract);
        Ok(())
    }

    fn load_balance(&self, address: &Address) -> Result<[u8; 32], String> {
        Ok(self
            .staged
            .balances
            .get(address)
            .copied()
            .unwrap_or([0; 32]))
    }

    fn store_balance(&mut self, address: Address, balance: [u8; 32]) -> Result<(), String> {
        self.staged.balances.insert(address, balance);
        Ok(())
    }

    fn commit(self) -> Result<(), String> {
        *self.parent = self.staged;
        Ok(())
    }
}
