
mod discovery;

use discovery::convert;

pub type AnyError = Box<dyn std::error::Error + Send + Sync>;
pub type Result<T> = std::result::Result<T, AnyError>;

fn help() {
    print!(r#"
        Convert arvados discovery document to a rust interface.

        Usage:
            aravods-api-generator <inputfile.json> <outputfile.rs>
    "#);
}



fn main() -> Result<()> {
    let mut args = std::env::args();
    args.next();
    let infile = args.next().unwrap_or_default();
    let outfile = args.next().unwrap_or_default();
    if infile == "" || outfile == "" {
        help();
        Ok(())
    } else {
        if let Ok(infd) = std::fs::File::open(&infile) {
            if let Ok(outfd) = std::fs::File::create(&outfile) {
                Ok(convert(infd, outfd)?)
            } else {
                Err(format!("File {} not created", outfile).into())
            }
        } else {
            Err(format!("File {} not found", infile).into())
        }
    }
}
