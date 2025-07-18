go run ./cmd/echoevm run -bin ./test/bins/build/Add_sol_Add.bin -function "add(uint256,uint256)" -args "1,2"

go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/Add.sol/Add.json -function "add(uint256,uint256)" -args "1,2"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/Sub.sol/Sub.json -function "sub(uint256,uint256)" -args "5,3"

go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/Fact.sol/Fact.json -function "fact(uint256)" -args "5"

go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/Require.sol/Require.json -function "test(uint256)" -args "1"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/Require.sol/Require.json -function "test(uint256)" -args "0"

