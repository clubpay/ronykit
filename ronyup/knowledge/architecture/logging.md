Log exclusively through x/telemetry/logkit (OpenTelemetry-bridged zap): inject *logkit.Logger via fx and use .With() for contextual fields; never import raw zap, slog, or log.
