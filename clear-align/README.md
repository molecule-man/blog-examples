## Running the benchmarks

```sh
GOEXPERIMENT=simd go test -run xxx -bench . -benchtime -count 8 -cpu 1 .
```

## Denoise

For the best possible reproducibility denoise your machine. Assuming intel:

```sh
# Assuming that you'll pin to cpu2

# --- find cpu2's SMT sibling (do this once) ---
cat /sys/devices/system/cpu/cpu2/topology/thread_siblings_list
# e.g. prints "2,14"  -> the sibling to offline is cpu14

# --- denoise ---
# 1. disable turbo (intel_pstate path; 1 = off)
echo 1 | sudo tee /sys/devices/system/cpu/intel_pstate/no_turbo

# 2. performance governor + EPP on every cpu
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/energy_performance_preference

# 3. offline the SMT sibling so cpu2 owns its physical core
echo 0 | sudo tee /sys/devices/system/cpu/cpu14/online

# 4. pin the uncore (ring + memory controller) to its hardware max
for d in /sys/devices/system/cpu/intel_uncore_frequency/package_*_die_*; do
  peak=$(cat "$d/initial_max_freq_khz")
  echo "$peak" | sudo tee "$d/max_freq_khz"
  echo "$peak" | sudo tee "$d/min_freq_khz"
done

# 5. transparent huge pages -> never (deterministic 4 KiB backing)
echo never | sudo tee /sys/kernel/mm/transparent_hugepage/enabled

# 6. run your benchmark pinned to cpu2
GOEXPERIMENT=simd taskset -c 2 go test -run xxx -bench . -benchtime -count 8 -cpu 1 .
```
