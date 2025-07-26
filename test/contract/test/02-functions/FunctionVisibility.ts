import { expect } from "chai";
import { ethers } from "hardhat";

describe("FunctionVisibility", function () {
  let functionVisibility: any;
  let owner: any;
  let user1: any;

  beforeEach(async function () {
    [owner, user1] = await ethers.getSigners();
    const FunctionVisibilityFactory = await ethers.getContractFactory("FunctionVisibility");
    functionVisibility = await FunctionVisibilityFactory.deploy();
    await functionVisibility.waitForDeployment();
  });

  describe("Public Functions", function () {
    it("should call public function", async function () {
      const result = await functionVisibility.publicFunction();
      expect(result).to.equal("Public function called");
    });

    it("should access public state variables", async function () {
      expect(await functionVisibility.publicValue()).to.equal(100);
    });
  });

  describe("Private Functions", function () {
    it("should call private function through public wrapper", async function () {
      const result = await functionVisibility.callPrivateFunction();
      expect(result).to.equal("Private function called");
    });

    it("should access private state variable through public getter", async function () {
      expect(await functionVisibility.getPrivateValue()).to.equal(200);
    });

    it("should set private state variable through public setter", async function () {
      await functionVisibility.setPrivateValue(300);
      expect(await functionVisibility.getPrivateValue()).to.equal(300);
    });
  });

  describe("Internal Functions", function () {
    it("should call internal function through public wrapper", async function () {
      const result = await functionVisibility.callInternalFunction();
      expect(result).to.equal("Internal function called");
    });

    it("should access internal state variable through public getter", async function () {
      expect(await functionVisibility.getInternalValue()).to.equal(300);
    });

    it("should set internal state variable through public setter", async function () {
      await functionVisibility.setInternalValue(400);
      expect(await functionVisibility.getInternalValue()).to.equal(400);
    });
  });

  describe("External Functions", function () {
    it("should call external function", async function () {
      const result = await functionVisibility.externalFunction();
      expect(result).to.equal("External function called");
    });
  });

  describe("State Variable Access", function () {
    it("should access all visibility levels of state variables", async function () {
      expect(await functionVisibility.publicValue()).to.equal(100);
      expect(await functionVisibility.getPrivateValue()).to.equal(200);
      expect(await functionVisibility.getInternalValue()).to.equal(300);
      expect(await functionVisibility.externalValue()).to.equal(400);
    });
  });
}); 