# Roadmap

## Phase 1: Interface Scaffold (DONE)
Define all interfaces, types, structs, and enums across 8 packages. No implementations. Validate that the type system compiles and packages compose correctly.

## Phase 2: Architecture Validation
Model concrete agent scenarios against the scaffold. Test how real problems compose against the Primitive interface. Identify gaps or friction in the algebra.

## Phase 3: Core Implementation
Implement the foundational runtime: EventBus routing, ResourceGovernor budget enforcement, PrimitiveRegistry with hot-swapping, and AgentInstance goroutine loop.

## Phase 4: Memory Hierarchy
Implement the L1-L5 memory stack with retrieval escalation and context window management.

## Phase 5: Type System
Integrate JSON Schema validation for structural types. Build semantic compatibility checking (likely LLM-backed).

## Phase 6: Coordination Patterns
Implement pipeline and mesh coordination patterns (doc 09). Add consolidator background process.

## Phase 7: JIT Crystallization
Reduce stable agent patterns to direct code paths for performance (doc 10).

## Long-term Vision
A system where agents can be composed, swapped, evolved, and reasoned about with the same rigor as algebraic expressions — while maintaining hard safety guarantees through the bedrock/governor layers.
