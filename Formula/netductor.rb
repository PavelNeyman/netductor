class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.70"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.70/netductor-darwin-arm64"
      sha256 "06cff992978fcfb916fb40e82c86dc7e8c9e3ff4d1cca5c6bad58bfcbe6efd27"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.70/netductor-darwin-amd64"
      sha256 "03e3281dec26c04866efc01401d9830b5e65551832343b3c448b8c712b7d24b6"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.70/netductor-linux-amd64"
      sha256 "a76ea7b233b5beb52e6f985e19bb2f441a74fba8c50f57e1677d1daca457ac57"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.70/netductor-linux-arm64"
      sha256 "a36ef46058e79f4e7fe76547b57fe82256d6fcbb4707a80c1fedcd53f8a95b00"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
