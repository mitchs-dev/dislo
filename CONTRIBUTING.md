# Contributing to Dislo

Thank you for your interest in contributing to Dislo! Your contributions are welcome and appreciated.

## How to Contribute

1. **Fork the Repository**
   - Click "Fork" at the top right of the [repository page](https://github.com/mitchs-dev/dislo).

2. **Clone Your Fork**
   - Clone your fork to your local machine:
     ```sh
     git clone https://github.com/yourusername/dislo.git
     cd dislo
     ```

3. **Create a Branch**
   - Create a new branch for your feature or bugfix:
     ```sh
     git checkout -b my-feature
     ```

4. **Make Your Changes**
   - Implement your changes, following the existing code style.
   - Add or update tests as appropriate.

5. **Test Your Changes**
   - Run tests locally to ensure nothing is broken:
     ```sh
     go build ./...
     go test ./...
     ```

6. **Commit and Push**
   - Commit your changes with a descriptive message:
     ```sh
     git add .
     git commit -m "Describe your change"
     git push origin my-feature
     ```

7. **Label Your Pull Request**
   - Before submitting your PR, add **exactly one** of the following labels:
     - `release:major`
     - `release:minor`
     - `release:patch`
   - This is required for automated versioning and release creation.
   - PRs without exactly one `release:` label cannot be merged.

8. **Open a Pull Request**
   - Go to your fork on GitHub and open a pull request against the `main` branch.
   - Fill out the pull request template.

## Guidelines

- Follow the MIT License.
- Write clear, concise commit messages.
- Update documentation if your changes affect usage or configuration.
- For large changes, consider opening an issue first to discuss your proposal.
- Ensure your code passes all tests and builds successfully.

## Code of Conduct

Please be respectful and considerate in all interactions. Dislo strives to be an inclusive and welcoming community.

## Need Help?

If you have questions, see [SUPPORT.md](http://_vscodecontentref_/0) or open an issue.

---

_Dislo is an open-source project licensed under the MIT License. See [LICENSE](http://_vscodecontentref_/1) for details._