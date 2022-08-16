import os

from typing import Literal, Final

import pyteal as pt
import beaker as bkr


VrfHash = pt.abi.StaticArray[pt.abi.Byte, Literal[64]]
VrfProof = pt.abi.StaticArray[pt.abi.Byte, Literal[80]]

class VrfIntake(bkr.Application):

    last_round: Final[bkr.ApplicationStateValue] = bkr.ApplicationStateValue(pt.TealType.uint64)
    randomness: Final[
        bkr.DynamicApplicationStateValue
    ] = bkr.DynamicApplicationStateValue(pt.TealType.bytes, max_keys=63)

    @bkr.create
    def create(self):
        return self.initialize_application_state()

    @bkr.delete(authorize=bkr.Authorize.only(pt.Global.creator_address()))
    def delete(self):
        return pt.Approve()

    @bkr.update(authorize=bkr.Authorize.only(pt.Global.creator_address()))
    def update(self):
        return pt.Approve()

    @bkr.external
    def noop(self):
        return pt.Approve()

    @bkr.external
    def ingest(self, round: pt.abi.Uint64, proof: VrfProof):
        return pt.Seq(
            # Only allow going forwards by 1 round at a time
            #pt.Assert(round.get() == self.last_round + pt.Int(1)),

            # Verify the proof
            verified := pt.VrfVerify.algorand(
                # msg is hash(round | seed)
                pt.Sha512_256(pt.Concat(pt.Itob(round.get()), pt.Block.seed(round.get()))),
                # proof
                proof.encode(),
                # pk
                pt.Txn.sender()
            ),
            # Make sure its valid
            pt.Assert(verified.output_slots[1].load() == bkr.consts.TRUE),
            # Add it to the global state
            self.insert_randomness(round.get(), verified.output_slots[0].load()),
            # Update last round
            self.last_round.set(round.get()),
        )

    @bkr.internal(pt.TealType.none)
    def insert_randomness(self, round, hash):
        return pt.Seq(
            (first_round := pt.ScratchVar(pt.TealType.bytes)).store(pt.Itob(round - pt.Int(63))),
            (to_delete := self.randomness[first_round.load()].get_maybe()),
            pt.If(to_delete.hasValue(), self.randomness[first_round.load()].delete()),
            self.randomness[pt.Itob(round)].set(hash)
        )


if __name__ == "__main__":
    app_id = None
    app_id_file = "../.app_id"

    from os.path import exists
    if exists(app_id_file):
        with open(app_id_file, "r") as f:
            app_id = int(f.read())

    v = VrfIntake()

    client = bkr.client.ApplicationClient(
        bkr.sandbox.get_algod_client(),
        v,
        signer=bkr.sandbox.get_accounts().pop().signer,
    )

    if app_id is None:
        app_id, app_addr, _ = client.create()
        print(f"Created app id: {app_id}")

        with open(app_id_file, "w") as f:
            f.write(str(app_id))

        client.fund(1 * bkr.consts.algo)
    else:
        client = client.prepare(app_id=app_id)
        client.update()

    import json
    with open("../contract.json", "w") as f:
        f.write(json.dumps(v.contract.dictify()))

    #sp = client.client.suggested_params()
    #sp.last = sp.first + 20
    #result = client.call(VrfIntake.get_seed, suggested_params=sp, round=10)
    #print(result.return_value)

