# PR Checklist for New Providers

Ensure the following 14 items are satisfied:

1. [ ] Target server docs retrieved and schema confirmed
2. [ ] Transport decided (Stdio vs. HTTP)
3. [ ] Auth method decided (Env var vs. URL query vs. Header)
4. [ ] Helper package (`pkg/<id>`) created with Key parser
5. [ ] Helper package tests passing
6. [ ] Provider struct implemented (`pkg/provider/<id>.go`)
7. [ ] RequiredCredentials defined accurately
8. [ ] GenerateConfig correctly maps to MCPConfig
9. [ ] Provider registered in `pkg/provider/registry.go`
10. [ ] Registry test updated (`pkg/provider/registry_test.go`)
11. [ ] Per-client adaptations added if necessary
12. [ ] QA scenarios written in `pkg/app/qa_scenarios_test.go`
13. [ ] `make test` and `make lint` completely green
14. [ ] `README.md` Provider Matrix updated