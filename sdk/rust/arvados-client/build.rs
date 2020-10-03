

const INPUT_FILENAME : &str = "arvados-api.json";
const OUTPUT_FILENAME : &str = "arvados-api.rs";

/// Call the arvados-api-generator crate to generate arvados-api.rs
/// This will be included in lib.rs.
/// OUT_DIR is target/build/arvados_client_<hash>
fn main() {
    use std::path::Path;
    let out_dir = std::env::var_os("OUT_DIR").unwrap();

    let dest_path = Path::new(&out_dir).join(OUTPUT_FILENAME);
    let src_path = Path::new(INPUT_FILENAME);

    println!("cargo:rerun-if-changed={}", INPUT_FILENAME);

    let src_file = std::fs::File::open(src_path).unwrap();
    let dest_file = std::fs::File::create(dest_path).unwrap();

    arvados_api_generator::convert(src_file, dest_file).unwrap();
}
