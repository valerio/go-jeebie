# go-jeebie

[![CI](https://github.com/valerio/go-jeebie/workflows/CI/badge.svg)](https://github.com/valerio/go-jeebie/actions)

A Game Boy emulator written in Go.

## Requirements

- Go 1.23 or later

## Building and Running

```bash
# Build the emulator
make build

# Run a Game Boy ROM with SDL2 (must have SDL2 installed)
make run-sdl2 path/to/rom.gb

# Run tests
make test

# Run all tests, including snapshot tests for Blargg's test suite
make test-all
```


## Status

Still a work in progress. Can currently run some simple games, and passes basic test roms for rendering/CPU behavior, see the [Test ROMs](#test-roms) section below.

### Games

Simple games running in the emulator:

![Tetris](screenshots/tetris.png) ![Super Mario Land](screenshots/super-mario-land.png)

### Test ROMs

ROMs are collected from the excellent [c-sp’s gameboy-test-roms collection](https://github.com/c-sp/gameboy-test-roms).
Huge thanks to the original authors (Blargg, Matt Currie and more) and maintainers of these suites.

These test ROMs are run as part of 
```bash
make test-integration
```

A snapshot of the screen is taken at the end of each test, and compared to a reference snapshot stored in `test/integration/testdata/snapshots`.

<details>
<summary>Passing Tests (with generated snapshots)</summary>

<!-- SNAPSHOTS:START -->
<table>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/01-special.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>01-special ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/02-interrupts.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>02-interrupts ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/03-op%20sp%2Chl.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>03-op sp,hl ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/04-op%20r%2Cimm.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>04-op r,imm ✅</sub></div></td>
  </tr>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/05-op%20rp.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>05-op rp ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/06-ld%20r%2Cr.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>06-ld r,r ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/07-jr%2Cjp%2Ccall%2Cret%2Crst.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>07-jr,jp,call,ret,rst ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/08-misc%20instrs.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>08-misc instrs ✅</sub></div></td>
  </tr>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/09-op%20r%2Cr.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>09-op r,r ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/10-bit%20ops.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>10-bit ops ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/11-op%20a%2C%28hl%29.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>11-op a,(hl) ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg-acid2.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg-acid2 ✅</sub></div></td>
  </tr>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_01-registers.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_01-registers ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_02-len_ctr.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_02-len_ctr ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_03-trigger.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_03-trigger ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_04-sweep.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_04-sweep ✅</sub></div></td>
  </tr>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_05-sweep_details.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_05-sweep_details ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_06-overflow_trigger.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_06-overflow_trigger ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_07-len_sweep_period_sync.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_07-len_sweep_period_sync ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_08-len_ctr_during_power.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_08-len_ctr_during_power ✅</sub></div></td>
  </tr>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/dmg_sound_09-wave_read_while_on.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>dmg_sound_09-wave_read_while_on ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/halt_bug.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>halt_bug ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/instr_timing.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>instr_timing ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/mem_timing_01-read.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>mem_timing_01-read ✅</sub></div></td>
  </tr>
  <tr>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/mem_timing_02-write.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>mem_timing_02-write ✅</sub></div></td>
    <td align="center"><div style="position:relative;display:inline-block;"><img src="test/integration/testdata/snapshots/mem_timing_03-modify.png" width="80" style="display:block;transition:transform 0.2s;" onmouseover="this.style.transform='scale(2)';this.style.zIndex='999';" onmouseout="this.style.transform='scale(1)';this.style.zIndex='1';" /><br><sub>mem_timing_03-modify ✅</sub></div></td>
    <td></td>
    <td></td>
  </tr>
</table>

<!-- SNAPSHOTS:END -->

</details>



## License

See the [license](./LICENSE.md) file for license rights and limitations (MIT).
