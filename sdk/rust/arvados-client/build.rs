


fn main() {
    use std::path::Path;
    let out_dir = std::env::var_os("OUT_DIR").unwrap();
    let dest_path = Path::new(&out_dir).join("arvados-api.rs");
    let src_path = Path::new("arvados-api.json");
    let src_file = std::fs::File::open(src_path).unwrap();
    let dest_file = std::fs::File::create(dest_path).unwrap();
    arvados_api_generator::convert(src_file, dest_file).unwrap();
}