# Homebrew Formula for Cleanup CLI
# To install: brew install --formula ./Formula/cleanup.rb

class Cleanup < Formula
  desc "Intelligent file organization CLI tool powered by local Ollama models"
  homepage "https://github.com/user/cleanup-cli"
  version "1.0.0"
  
  # For local installation, use file:// URL
  # url "file://#{Dir.pwd}/build/cleanup-1.0.0-darwin.tar.gz"
  # sha256 "..." # Calculate with: shasum -a 256 build/cleanup-1.0.0-darwin.tar.gz
  
  # For GitHub releases, use:
  # url "https://github.com/user/cleanup-cli/releases/download/v1.0.0/cleanup-1.0.0-darwin.tar.gz"
  # sha256 "..."
  
  depends_on "ollama" => :recommended
  
  def install
    bin.install "cleanup"
    
    # Install example config
    (share/"cleanup").install "cleanuprc.yaml.example"
    
    # Install documentation
    doc.install "README.md"
    
    # Install demo script
    (share/"cleanup").install "demo.sh" if File.exist?("demo.sh")
  end
  
  def post_install
    # Create config directory
    (var/"cleanup").mkpath
    
    # Copy example config if user doesn't have one
    config_file = "#{Dir.home}/.cleanuprc.yaml"
    unless File.exist?(config_file)
      cp "#{share}/cleanup/cleanuprc.yaml.example", config_file
      ohai "Configuration file created at #{config_file}"
    end
  end
  
  def caveats
    <<~EOS
      Cleanup CLI has been installed!
      
      Quick Start:
        1. Ensure Ollama is running:
           $ ollama serve
        
        2. Pull the required model:
           $ ollama pull llama3.2
        
        3. Run cleanup:
           $ cleanup
        
        4. Or organize a specific directory:
           $ cleanup organize ~/Downloads
      
      Configuration:
        Edit: ~/.cleanuprc.yaml
      
      Documentation:
        Run: cleanup --help
        View: #{doc}/README.md
    EOS
  end
  
  test do
    system "#{bin}/cleanup", "--help"
  end
end
