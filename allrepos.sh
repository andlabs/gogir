set -e
ls /usr/share/gir-1.0/ | sed 's/-.*$//g' | while read -r repo; do
	echo $repo
	G_DEBUG=fatal-warnings ./gogir $repo > /dev/null
done
