# Solidity Syntax Features Demo

This project demonstrates all major Solidity syntax features and language constructs through focused, single-purpose contracts.

## Project Structure

The contracts are organized by syntax feature categories:

### 01-data-types/
- **BoolType.sol** - Boolean operations and logical operators
- **IntegerTypes.sol** - Integer arithmetic and operations
- **AddressType.sol** - Address operations and low-level calls
- **BytesType.sol** - Fixed and dynamic bytes operations
- **StringType.sol** - String manipulation and operations
- **ArrayTypes.sol** - Fixed-size, dynamic, and multi-dimensional arrays
- **MappingTypes.sol** - Mapping operations and nested mappings
- **StructTypes.sol** - Struct definitions and operations
- **Add.sol** - Simple addition function
- **Sub.sol** - Simple subtraction function
- **Fact.sol** - Recursive factorial calculation

### 02-functions/
- **FunctionVisibility.sol** - Public, private, internal, external visibility
- **FunctionMutability.sol** - Pure, view, payable, and state-changing functions

### 03-control-flow/
- **IfElse.sol** - Conditional statements and ternary operators
- **Loops.sol** - For, while, do-while loops with break and continue
- **Switch.sol** - Enum-based conditional branching
- **Require.sol** - Require statement for condition checking

### 04-modifiers/
- **CustomModifiers.sol** - Custom modifier creation and usage

### 05-events/
- **EventExamples.sol** - Event declaration, emission, and indexing
- **Lock.sol** - Time-locked contract with withdrawal events

### 06-inheritance/
- **BaseContract.sol** - Base contract with virtual functions and modifiers
- **ChildContract.sol** - Contract inheritance and function overriding

### 07-libraries/
- **MathLibrary.sol** - Library with utility functions
- **LibraryUser.sol** - Contract using library functions

### 08-advanced/
- **Assembly.sol** - Inline assembly operations

## Running Tests

```bash
# Compile contracts
npx hardhat compile

# Run all tests
npx hardhat test

# Run specific test category
npx hardhat test test/01-data-types/
```

## Features Demonstrated

### Data Types
- Boolean operations (AND, OR, NOT)
- Integer arithmetic (+, -, *, /, %, **)
- Address operations (transfer, send, call, delegatecall)
- Bytes manipulation (fixed-size, dynamic, concatenation)
- String operations (concatenation, conversion)
- Array operations (fixed, dynamic, multi-dimensional)
- Mapping operations (simple, nested, with structs)
- Struct operations (nested structs, arrays, mappings)
- Basic math functions (add, subtract, factorial)

### Functions
- Visibility modifiers (public, private, internal, external)
- Mutability modifiers (pure, view, payable)
- Function overloading and overriding

### Control Flow
- Conditional statements (if, else, else if)
- Ternary operators
- Loops (for, while, do-while)
- Break and continue statements
- Require statements for validation
- Enum-based conditional branching

### Advanced Features
- Custom modifiers
- Events and indexing
- Contract inheritance
- Library usage
- Inline assembly
- Time-locked contracts

Each contract focuses on a specific syntax feature, making it easy to understand and learn individual Solidity concepts.
