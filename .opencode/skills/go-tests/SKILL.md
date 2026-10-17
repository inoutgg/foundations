---
name: go-tests
description: Use when creating, editing, or rewriting Go tests in this repository; enforce top-level test naming, subtest descriptions, and parallel execution conventions.
---

# Go tests

- Name every top-level test function `Test<StructName>_<MethodName>` or `Test<StructName>_<MethodName>_<ComplexTest>`. Use the optional suffix only for a distinct complex test scenario.
- Put all cases below the corresponding top-level test in `t.Run(...)` subtests. Name each subtest `should <expected technical/business behaviour> when <condition>` or `must <expected technical/business behaviour> when <condition>`.
- Prefer `t.Parallel()` for both top-level tests and subtests whenever they can run independently. Do not parallelize tests that mutate shared state or otherwise require sequential execution.

Example:

```go
func TestPool_Begin(t *testing.T) {
	t.Parallel()

	t.Run("should start a transaction when the pool is available", func(t *testing.T) {
		t.Parallel()
		// Arrange, act, and assert.
	})
}
```
