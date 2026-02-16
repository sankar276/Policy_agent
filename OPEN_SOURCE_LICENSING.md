# Open Source Licensing - Complete! ✅

The Policy AI Agent is now fully open source under the **MIT License**, allowing anyone to freely use, modify, and distribute the code.

## What Was Added

### 1. **MIT License Files** ✅

Created LICENSE files in all project directories:
- [LICENSE](LICENSE) - Root license
- [policy-agent/LICENSE](policy-agent/LICENSE) - Go implementation
- [policy-agent-py/LICENSE](policy-agent-py/LICENSE) - Python implementation

**MIT License allows:**
- ✅ Commercial use
- ✅ Modification
- ✅ Distribution
- ✅ Private use

**Requirements:**
- Include copyright notice
- Include license text

### 2. **Contributing Guidelines** ✅

Created [CONTRIBUTING.md](CONTRIBUTING.md) with:
- Code of conduct
- Development setup instructions
- Coding standards (Go, Python, Rego)
- Pull request process
- Testing guidelines
- Instructions for adding new validators

### 3. **Notice File** ✅

Created [NOTICE](NOTICE) file acknowledging:
- Open source dependencies
- Third-party libraries used
- Claude AI by Anthropic
- Open Policy Agent (OPA)

### 4. **README Badges** ✅

The main [README.md](README.md) already includes:
```markdown
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
```

## License Details

### MIT License Summary

```
Copyright (c) 2025 Policy AI Agent Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software...
```

**What this means:**
- Anyone can use this code for any purpose
- Anyone can modify and distribute it
- No warranty is provided
- Original license must be included in distributions

### Dependencies

All dependencies are also open source:

**Go Implementation:**
- Open Policy Agent (OPA) - Apache 2.0
- Anthropic Go SDK - MIT
- Cobra - Apache 2.0
- Viper - MIT

**Python Implementation:**
- Anthropic Python SDK - MIT
- Click - BSD 3-Clause
- PyYAML - MIT

## For Users

### How to Use This Project

1. **Clone or fork the repository:**
   ```bash
   git clone https://github.com/your-org/policy-agent.git
   ```

2. **Use as-is or modify:**
   - Use the binaries/packages directly
   - Modify the code for your needs
   - Integrate into your products

3. **Include the license:**
   - Keep the LICENSE file in your distribution
   - Include copyright notice in documentation

### Commercial Use

✅ **You can:**
- Use in commercial products
- Sell services based on this code
- Integrate into proprietary software
- Modify without sharing changes (though we'd love contributions!)

❌ **You cannot:**
- Remove the copyright notice
- Sue contributors for warranty issues
- Use contributor names for endorsement without permission

## For Contributors

### How to Contribute

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed guidelines.

**Quick start:**

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

**By contributing, you agree:**
- Your contributions will be licensed under MIT
- You have the right to submit the contribution
- You're okay with the code being used commercially

### Code of Conduct

We are committed to providing a welcoming community:
- Be respectful and constructive
- Welcome newcomers
- Focus on what's best for the project
- Show empathy towards others

## Publishing to Package Registries

### Go

Can be imported directly from GitHub:
```go
import "github.com/your-org/policy-agent/internal/validator"
```

### Python (PyPI)

To publish to PyPI:

```bash
cd policy-agent-py

# Build package
python -m build

# Upload to PyPI
python -m twine upload dist/*
```

Then others can install:
```bash
pip install policy-agent
```

## Attribution

When using this project, we appreciate (but don't require) attribution:

```
This project uses Policy AI Agent (https://github.com/your-org/policy-agent)
```

Or in documentation:
```markdown
Powered by [Policy AI Agent](https://github.com/your-org/policy-agent) -
MIT Licensed
```

## Questions?

### About Licensing

- **Q: Can I use this in my company's internal tools?**
  - A: Yes! MIT allows commercial use.

- **Q: Do I need to share my modifications?**
  - A: No, but contributions are welcome!

- **Q: Can I sell products based on this?**
  - A: Yes! MIT allows commercial use.

- **Q: Do I need to include the license?**
  - A: Yes, you must include the MIT license text.

### About Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) or open an issue.

## Summary

The Policy AI Agent is now:

✅ **Fully open source** under MIT License
✅ **Free to use** for any purpose
✅ **Modifiable and redistributable**
✅ **Ready for contributions** with clear guidelines
✅ **Commercially friendly** with no restrictions

---

**Thank you for using and contributing to Policy AI Agent!** 🚀

If you find this project useful, please consider:
- ⭐ Starring the repository
- 🐛 Reporting bugs
- 💡 Suggesting features
- 🤝 Contributing code
- 📢 Sharing with others

## Resources

- [LICENSE](LICENSE) - Full license text
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [NOTICE](NOTICE) - Dependency acknowledgments
- [README.md](README.md) - Project overview
- [Choose a License](https://choosealicense.com/licenses/mit/) - MIT License explained
