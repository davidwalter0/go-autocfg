/*
Package autocfg allows pointing configuration data to config option settings
matching your work environment

# Overview

autocfg looks for it's configuration in one of 3 places in the following order:

 1. A path named in the environment variable AUTOCFG_FILENAME
 2. .autocfg.json in the current working directory
 3. ~/.config/{{program name}}/autocfg.json, where {{program name}} is
    path.Base(os.Args[0]), path.Ext(os.Args[0]))

The configuration if found can be loaded directly from path returned
by FindConfiguration()
*/
package autocfg
