class Netductor < Formula
  desc "Netductor operator (Mac/Linux CLI+TUI+Web)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.129"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.129/netductor-op-darwin-arm64"
      sha256 "15bde3ac84565421c531533cafe46591769d52a2fb93677d73f5809b6a2cc4b0"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.129/netductor-op-linux-amd64"
      sha256 "d698e86b9f30ac1d69c5d8e5f70053d747fbe87e34602c032629329b86597400"
    end
  end

  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version 2>&1")
  end
end
