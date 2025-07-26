import {
  time,
  loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import hre from "hardhat";

describe("Fact", function () {
  async function deployFactFixture() {
    const [owner, otherAccount] = await hre.ethers.getSigners();

    const Fact = await hre.ethers.getContractFactory("Fact");
    const fact = await Fact.deploy();

    return { fact, owner, otherAccount };
  }

  describe("Deployment", function () {
    it("Should set the right result", async function () {
      const { fact } = await loadFixture(deployFactFixture);
      const ret = await fact.fact(5);
      expect(ret).to.equal(120);
    });
  });
});
