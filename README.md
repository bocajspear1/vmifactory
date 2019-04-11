# VMIFactory - VM Image Factory

The VMIFactory helps automate and provide access to built and easily importable VM images for different hypervisors. Using [Packer](https://www.packer.io), VMIFactory periodically updates existing images to provide a range of updated, ready-to-import images.

### Workflow

1. **Create initial images for system** - Create a base image of your system for each hypervisor. They each need their own tools and formats, so its easier to do this instead of trying to convert the image later. (You might be able to convert them now, work out any issues, then use the resulting images though)
2. **Create the image directory tree** - Use the builder script to create a template image directory tree:
* `<image-name>`- Lowercase, `-` separated name of the image
    * `<image-name>.json` - A json file that defines the image hypervisors, configuration and metadata.
    * `run` - A directory. All scripts in this directory will be executed in alphabetical order **EACH** time the image is rebuilt with Packer. Use this for things like updating applications and such.
    * `runonce` - A directory. All scripts in this directory will be executed **ONCE** then placed in the `used` directory. Use this for adding new applications to the images and single time commands.
        * `used` - A directory containing used scripts, they will be their original name with the timestamp executed attached to them.
