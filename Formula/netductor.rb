class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.67"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.67/netductor-darwin-arm64"
      sha256 "02ccfa7da7eeb29902ab087cb584adb3db263b61e364cb3f5cd42927b6ee22e2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.67/netductor-darwin-amd64"
      sha256 "9d3606c67820333ba1105e312cb7550865140c18a20d48d130cd0a4c1e592571"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.67/netductor-linux-amd64"
      sha256 "b97582fcc5fde99d1b40fdbc9882c08427267f1818f3a66748ba6eada59dd8e4"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.67/netductor-linux-arm64"
      sha256 "0a3f4ce77c9f8dba93dd7442dd32577bcb757626e000543b8de8ec3ea19391f8"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
