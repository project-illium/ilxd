fn main() {
    // Check if the target OS is macOS
    if cfg!(target_os = "macos") {
        println!("cargo:rustc-link-lib=framework=SystemConfiguration");
        // Add any other macOS-specific frameworks here
    }
    // Additional build script code (if any) goes here...
}