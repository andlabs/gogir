set -e
# JSCore is being dumb so omit that for now (TODO)
# win32 doesn't exist as a repo?! omit that too for now (TODO)
ls /usr/share/gir-1.0/ | sed 's/-/ /g' | egrep -v 'JSCore|win32' | sed 's/\.gir$//g' | while read -r repo ver; do
	echo $repo $ver
	G_DEBUG=fatal-warnings ./gogir $repo $ver > /dev/null
done
