# Next Steps

## Immediate

- [ ] Model concrete problems against the interface scaffold to validate architecture self-organization
- [ ] Write `integration_test.go` — end-to-end test exercising type composition across packages
- [ ] Decide on JSON Schema validation library (`santhosh-tekuri/jsonschema/v6` proposed in plan, only needed in `types/`)

## Implementation (after validation)

- [ ] Implement `EventBus.Route` with priority channel routing
- [ ] Implement `ResourceGovernor.Execute` with budget checks and degradation
- [ ] Implement `PrimitiveRegistry` with type-checked `Swap` and bedrock protection
- [ ] Implement `AgentInstance.Run` with two-phase select pattern
- [ ] Implement `ContextWindow` as L1 memory store
- [ ] Implement `RetrievalStack` with level escalation

## Future

- [ ] Consolidator — background goroutine for memory consolidation between sessions
- [ ] JIT crystallization — reducing stable agent patterns to direct code (doc 10)
- [ ] Profiling vs evals framework (doc 16)
- [ ] Model selection integration (doc 17)
