class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.68"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.68/netductor-darwin-arm64"
      sha256 "05f7f0c7474e8c00eeb14e3d0aba0ee556f67084480cc4ab098ccf2d7b98bc27"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.68/netductor-darwin-amd64"
      sha256 "db743beb0af887bc291d31ad11251707fcec52eb0db0578a58610efb33eb0619"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.68/netductor-linux-amd64"
      sha256 "703dc614f5cf4c038c77e8f9196f08f193afea65da30452f8e3b07c010865dd3"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.68/netductor-linux-arm64"
      sha256 "faba93da5874aee25f06f31b2d5ac7e3f51cca7480ff0fc5330d5dcca2a3a405"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
