import {
  time,
  loadFixture,
} from "@nomicfoundation/hardhat-toolbox/network-helpers";
import { anyValue } from "@nomicfoundation/hardhat-chai-matchers/withArgs";
import { expect } from "chai";
import hre from "hardhat";

describe("Add", function () {
  async function deployAddFixture() {
    const [owner, otherAccount] = await hre.ethers.getSigners();

    const Add = await hre.ethers.getContractFactory("Add");
    const add = await Add.deploy();

    return { add, owner, otherAccount };
  }

  describe("Deployment", function () {
    it("Should set the right result", async function () {
      const { add } = await loadFixture(deployAddFixture);
      const ret = await add.add(1, 2);
      expect(ret).to.equal(3);
    });
  });
});
