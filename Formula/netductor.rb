class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.4-dev"
  license "MIT"

  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "a4a77719e8deb789abeceedb88bb3eba93e426df4bc42a656a20c108245731f3"

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "4bb8458698e88bb01db72f78566f344f1aae825e657a880d581656da7f4474be"
    end
  end

  def install
    bin.install Dir["netductor*"].first => "netductor"
  end

  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
