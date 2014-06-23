set -e
# JSCore is being dumb so omit that for now (TODO)
# win32 doesn't exist as a repo?! omit that too for now (TODO)
# AppStream fails with ** (process:5348): CRITICAL **: g_irepository_find_by_name: assertion 'typelib != NULL' failed despite the typelib being there (TODO)
# BraseroBurn has a version mismatch (TODO)
# FolksEds and FolksTelepathy also have the typelib != NULL failure (TODO; TODO are the typelibs there?)
# Grip and Hud complain about the typeinfo not being found despite having the packages installed (TODO)
# UnityExtras: see FolksEds above (TODO)
# Urfkill: see Grip above
ls /usr/share/gir-1.0/ | sed 's/-/ /g' |
	egrep -v 'JSCore|win32|AppStream|BraseroBurn|FolksEds|FolksTelepathy|Grip|Hud|UnityExtras|Urfkill' |
	sed 's/\.gir$//g' | while read -r repo ver; do
		echo $repo $ver 1>&2
		if [ Z$1 != Z ]; then
			G_DEBUG=fatal-warnings ./gogir $repo $ver $1
		else
			G_DEBUG=fatal-warnings ./gogir $repo $ver json > /dev/null
		fi
	done

# some girs I would not install due to unwanted or unverified dependencies:
# - emscritem (wants to pull in java)
# - ibus-anthy-dev (wants to pull in ibus)
# - libaccount-plugin-1.0-dev (wants to pull in signond)
# - libcryptui-dev (wants to pull in seahorse-daemon)
# - libfriends-dev, libfriends-gtk-dev (want to pull in friends-dispatcher)
# - libguestfs-gobject-dev (wants to pull in a lot of things, including btrfs)
# - libmuffin-dev (wants to pull in muffin-common)
# - libsignon-glib-dev (wants to pull in signond)
# - libskk-dev (wants to pull in skkdic)
# - libunity-webapps-dev (wants to pull in a bunch of unity stuff)
# - ubiquity-frontend-gtk (not a problem itself; pulls in a bunch of stuff I don't know what it is and would need to check)
# this may change in the future
