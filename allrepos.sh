set -e
ls /usr/share/gir-1.0/ | sed 's/-/ /g' | sed 's/\.gir$//g' | while read -r repo ver; do
	echo $repo $ver
	G_DEBUG=fatal-warnings ./gogir $repo $ver > /dev/null
done
