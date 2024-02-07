# Music-Utils

Original Repo and work done by https://github.com/Zibbp/music-utils.
Just fixed the Dockerfile and updated everything to current versions.

## Usage

```sh
  -save-spotify
        Save Spotify playlists to files
  -to-tidal
        Imports Spotify playlists to Tidal
 -import-navidrome
        Generates Navidrome playlist files from Tidal using Navidrome's database
  -save-tidal
        Save provided Tidal playlists to files
  -process-lidarr-wanted
        Find wanted Lidarr albums on Tidal and save to file
```

## Setup

1. Modify the `docker-compose.yml` file.
2. Fill in the needed keys into the docker-compose file. CARE: don´t use any quotation marks for the keys, just the blank strings.
3. Edit the `docker-compose.yml` with the command you want to run.

## Notes

Attempting to find music between platforms proved to be quite difficult. Tidal does not have an ISRC endpoint leaving me to search track by Title - Artist or Title - Album which can fail due to slight differences in naming between platforms. Any tracks not found during any steps are saved to a file within the `data` directory. A majority of the time these tracks do exist but has a difference causing it to be not found.