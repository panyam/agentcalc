# Next Steps

## Immediate

- [ ] Write kernel tests (`chakra_test.go`, `registry_test.go`, `store_test.go`)
  - Test ScopedStore isolation and delta merging
  - Test DeltaGate rejection (namespace enforcement)
  - Test InvokeAsync with cancellation and write observation
- [ ] Build 4 SWE-bench tool primitives: ReadFile, SearchCode, EditFile, RunTests
- [ ] Build ReActAgent primitive (for loop that captures a Registry)
- [ ] Build BudgetGate (global gate for budget enforcement)
- [ ] Build NamespaceGate (DeltaGate for store write namespace enforcement)
- [ ] Add reflexivity: after each tool call, write Hint to `run:hints` in Store

## Run

- [ ] Run on 10 SWE-bench Lite instances
- [ ] Read the hints — they tell you what to build next

## Design Considerations (Not Urgent)

- [ ] Runtime topology traceability — Connect/Disconnect calls should be observable
- [ ] Store write atomicity — batch merge for concurrent async children (if needed after real use)
- [ ] Update examples to use ConnectWithDelta for namespace enforcement

## After Running On Real Problems

- [ ] Let the hints and traces tell you what's missing
- [ ] Consult `docs/v0/` reference map when you hit specific pain
- [ ] Add things one at a time, justified by actual failures
