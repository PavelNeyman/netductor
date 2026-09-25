class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.7"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.7/netductor-op-darwin-arm64"
      sha256 "64be1c7b0dc91f46742950f4258f7b85bce9315242f5ba97dc70cd3ac1f44d69"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.7/netductor-op-darwin-amd64"
      sha256 "9f134678d164cf7aa9826826c5cc266b265404e0d3bd8607d98a37b3381b76cc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.7/netductor-op-linux-amd64"
      sha256 "03a651ce79f91ea3182ab9d5e281dedc57bd62ee0bb3881bf7b1091a98830b39"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.7/netductor-op-linux-arm64"
      sha256 "5441435638408196fa8bf67f4dee75372b17cc1c8664d9ad08d3cd3d7b3653bb"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
