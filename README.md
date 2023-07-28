# safeopen

**This is not an officially supported Google product.**

Safe-by-construction library with file open/create primitives for Golang that
are not vulnerable to path traversal attacks. The library supports Unix and
Windows systems. OS native safe primitives are leveraged where available (e.g.
openat2 + RESOLVE_BENEATH). Symbolic links are followed only if there is a safe
way to prevent traversal (e.g. on platforms where OS level safe primitives are
available), otherwise an error is returned.

## Usage

All these library functions expect a base directory as their first parameter.
There are two families of API functions, they have the suffix:

-   #1 At: The file to be opened must be directly in the base directory
-   #2 Beneath: The file to be opened must be somewhere underneath the base
    directory

Example:

```
    fd, err := safeopen.OpenBeneath("/workdir", filenameFromUserInput)
    if err != nil {
        return fmt.Errorf("unable to open file %v: %v", filenameFromUserInput, err)
    }
  // now do the same what you would with the return value of `os.Open`
  ...
```

The library also supports replacement functions of `os.ReadFile` and
`os.WriteFile`. Example:

```
    data, err := safeopen.ReadFileBeneath("/workdir", filenameFromUserInput)
    if err != nil {
        return fmt.Errorf("unable to open file %v: %v", filenameFromUserInput, err)
    }
  // now you can process data safely
  ...
```
