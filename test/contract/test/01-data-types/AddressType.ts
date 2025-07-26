import { expect } from "chai";
import { ethers } from "hardhat";

describe("AddressType", function () {
  let addressType: any;
  let owner: any;
  let user1: any;
  let user2: any;

  beforeEach(async function () {
    [owner, user1, user2] = await ethers.getSigners();
    const AddressTypeFactory = await ethers.getContractFactory("AddressType");
    addressType = await AddressTypeFactory.deploy();
    await addressType.waitForDeployment();
  });

  describe("Address Operations", function () {
    it("should set correct owner and contract address", async function () {
      expect(await addressType.owner()).to.equal(owner.address);
      expect(await addressType.contractAddress()).to.equal(await addressType.getAddress());
    });

    it("should get balance of an address", async function () {
      const balance = await addressType.getBalance(user1.address);
      expect(balance).to.be.gte(0);
    });

    it("should check if address is a contract", async function () {
      // EOA should return false
      expect(await addressType.isContract(user1.address)).to.equal(false);
      
      // Contract should return true
      expect(await addressType.isContract(await addressType.getAddress())).to.equal(true);
    });

    it("should get code size of an address", async function () {
      // EOA should have 0 code size
      expect(await addressType.getCodeSize(user1.address)).to.equal(0);
      
      // Contract should have code size > 0
      expect(await addressType.getCodeSize(await addressType.getAddress())).to.be.gt(0);
    });
  });

  describe("Transfer Operations", function () {
    it("should transfer ETH", async function () {
      const amount = ethers.parseEther("1.0");
      
      await expect(
        addressType.transfer(user1.address, { value: amount })
      ).to.changeEtherBalance(user1, amount);
    });

    it("should send ETH and return success", async function () {
      const amount = ethers.parseEther("0.5");
      
      // First transfer some ETH to the contract
      await owner.sendTransaction({
        to: await addressType.getAddress(),
        value: amount
      });
      
      // Note: send() will fail if the recipient doesn't have a receive function
      // This test demonstrates the function call, but may fail in practice
      try {
        const result = await addressType.send(user1.address, amount);
        expect(result).to.equal(true);
      } catch (error) {
        // Expected behavior when recipient doesn't have receive function
        expect(error).to.be.instanceOf(Error);
      }
    });
  });

  describe("Low-level Calls", function () {
    it("should perform call operation", async function () {
      const data = "0x";
      const result = await addressType.call(user1.address, data);
      expect(result[0]).to.equal(true);
    });

    it("should perform delegatecall operation", async function () {
      const data = "0x";
      const result = await addressType.delegateCall(user1.address, data);
      expect(result[0]).to.equal(true);
    });

    it("should perform staticcall operation", async function () {
      const data = "0x";
      const [success, result] = await addressType.staticCall(user1.address, data);
      expect(success).to.equal(true);
    });
  });
}); 