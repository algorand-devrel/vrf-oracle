VRF Demo
--------

WORK IN PROGRESS



Create the app in `contract/` with 
```sh
cd contract
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python contract.py
```

Clone go-algorand to and run `make install`

Tweak go.mod to point to the version you just built (needs libsodium)

run 
```sh
go run main.go
```

and it should start w
