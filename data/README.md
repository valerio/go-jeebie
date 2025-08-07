# Data Directory

This directory contains static data files used by the emulator:

## opcodes.json

Game Boy instruction set data sourced from [lmmendes/game-boy-opcodes](https://github.com/lmmendes/game-boy-opcodes). 

This file is used by the code generator in `jeebie/disasm/` to create the disassembly lookup tables. It contains:

- **unprefixed**: Regular Game Boy CPU instructions (0x00-0xFF)
- **cbprefixed**: CB-prefixed extended instructions (0x00-0xFF with CB prefix)

Each instruction includes:
- Mnemonic (assembly instruction name)
- Length in bytes
- Cycle timing information
- CPU flag effects
- Operand information

The data is used to generate `jeebie/disasm/generated.go` which contains the lookup tables for the live disassembly feature.