/*
Package autocfg allows pointing configuration data to config option settings
matching your work environment

Overview.

When DirectUnionMode is set autocfg priorizes configuration loading
each of the files in the following list each file's values replace any
previously found and unmarshaled files.

- /etc/{{program-name}}/config.json
- ~/.config/{{program-name}}/config.json
- .{{program-name}}.json in the current directory
- When set a file named in the environment variable AUTOCFG_FILENAME

The last file found has priority or dominates prior configurations
loaded.

When DirectFirstFoundMode is set autocfg loads the first found
configuration and stops loading when a file is found.

- When set a file named in the environment variable AUTOCFG_FILENAME
- .{{program-name}}.json in the current directory
- ~/.config/{{program-name}}/config.json
- /etc/{{program-name}}/config.json

When DirectAndIndirectMode is set then search DirectFirstFoundMode. If
no configuration is found then search indirect autocfg files in the
following 3 places and load from the first file found. For an indirect
auto config is performed the following order:

 1. A path named in the environment variable AUTOCFG_FILENAME
 2. .autocfg.json in the current working directory
 3. ~/.config/{{program name}}/autocfg.json, where {{program name}} is
    path.Base(os.Args[0]), path.Ext(os.Args[0]))

The configuration if found can be loaded directly from the path
returned by FindConfiguration()

The order of evaluation of configuration options follows this sequence.

1. file - Files must be created and saved prior to execute. When a
configuration file is found, load and unmarshal to the app object
supplied as the argument to the configuration call.

2. env - Environment variables are static pre-runtime; but may precede
the execution call, when an env variable is set, use that value and
replace an existing value(s) option specified in a file loaded
configuration in 1.

3. flag - Flags are evaluated from the command line. When flags are
specified, set corresponding object members from command line flag
argument and replace option specified in 1. or 2.

*/

package autocfg
