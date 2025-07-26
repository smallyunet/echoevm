import {
  time,
  loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import hre from "hardhat";

describe("Require", function () {
  async function deployRequireFixture() {
    const [owner, otherAccount] = await hre.ethers.getSigners();

    const Require = await hre.ethers.getContractFactory("Require");
    const require = await Require.deploy();

    return { require, owner, otherAccount };
  }

  describe("Deployment", function () {
    it("Should set the right result", async function () {
      const { require } = await loadFixture(deployRequireFixture);
      await expect(require.test(1)).to.be.fulfilled;
      await expect(require.test(0)).to.be.revertedWith("a must be greater than 0");
    });
  });
});
