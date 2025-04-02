## Release Process

jsonapi can be released as often as needed for other releases, such as [go-tfe](https://github.com/hashicorp/go-tfe).

### Preparing a release

First make sure that the CHANGELOG.md is up to date with all the changes included since the last release. You can find out what changes were made since the last release by comparing the main branch with the last release tag. ([Example](https://github.com/hashicorp/jsonapi/compare/v1.4.1...main))

Steps to prepare the changelog for a new release:

1. Replace `# Unreleased` with the version you are releasing.
2. Ensure there is a line with `# Unreleased` at the top of the changelog for future changes. Ideally we don't ask authors to add this line; this will make it clear where they should add their changelog entry.
3. Ensure that each existing changelog entry for the new release has the author(s) attributed and a pull request linked, i.e `- Some new feature/bugfix by @some-github-user (#3)[link-to-pull-request]`
4. Open a pull request with these changes titled `vX.XX.XX Changelog`. Once approved and merged, you can go ahead and create the release.


### Creating a release

1. [Create a new release in GitHub](https://help.github.com/en/github/administering-a-repository/creating-releases) by clicking on "Releases" and then "Draft a new release"
2. Set the `Tag version` to a new tag, using [Semantic Versioning](https://semver.org/) as a guideline.
3. Set the `Target` as `main`.
4. Set the `Release title` to the tag you created, `vX.Y.Z`
5. Use the description section to describe why you're releasing and what changes you've made. You can copy-paste the changelog entries that belong to the new release. You can also use the following headers in the description of your release:
   - BREAKING CHANGES: Use this for any changes that aren't backwards compatible. Include details on how to handle these changes.
   - FEATURES: Use this for any large new features added,
   - ENHANCEMENTS: Use this for smaller new features added
   - BUG FIXES: Use this for any bugs that were fixed.
   - NOTES: Use this section if you need to include any additional notes on things like upgrading, upcoming deprecations, or any other information you might want to highlight.

   Example:

   ```markdown
   FEATURES
   * Adds NullableRelationship generic to assist with encoding explicit null or empty values by @some-github-user (#3)[link-to-pull-request]

   BUG FIXES
   * Fix description of a bug by @some-github-user (#2)[link-to-pull-request]
   ```

6. Don't attach any binaries. The zip and tar.gz assets are automatically created and attached after you publish your release.
7. Click "Publish release" to save and publish your release.