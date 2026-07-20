//! Codec implementations (G.711 / G.722 / G.729 / Opus / telephone-event).

pub mod g722;
pub mod g729;
#[cfg(feature = "opus")]
pub mod opus;
pub mod pcma;
pub mod pcmu;
pub mod telephone_event;
