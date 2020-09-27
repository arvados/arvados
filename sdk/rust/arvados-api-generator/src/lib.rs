//!
//! Library to convert google discovery to the arvados rust api.
//!

mod discovery;
pub use discovery::convert;

pub type AnyError = Box<dyn std::error::Error + Send + Sync>;
pub type Result<T> = std::result::Result<T, AnyError>;
