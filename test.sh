go run ./cmd/echoevm run -bin ./test/bins/build/Add_sol_Add.bin -function "add(uint256,uint256)" -args "1,2"
go run ./cmd/echoevm run -artifact ./test/contract/artifacts/contracts/Add.sol/Add.json -function "add(uint256,uint256)" -args "1,2"

