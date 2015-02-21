package ofp13

import (
	"bytes"
	"io"

	"github.com/netrack/openflow/encoding/binary"
)

const (
	MT_STANDARD MatchType = iota
	MT_OXM
)

// MatchType indicates the match structure (set of fields that compose the
// match) in use. The match type is placed in the type field at the beginning
// of all match structures.
type MatchType uint16

const (
	XMT_OFB_IN_PORT OXMField = iota
	XMT_OFB_IN_PHY_PORT
	XMT_OFB_METADATA
	XMT_OFB_ETH_DST
	XMT_OFB_ETH_SRC
	XMT_OFB_ETH_TYPE
	XMT_OFB_VLAN_VID
	XMT_OFB_VLAN_PCP
	XMT_OFB_IP_DSCP
	XMT_OFB_IP_ECN
	XMT_OFB_IP_PROTO
	XMT_OFB_IPV4_SRC
	XMT_OFB_IPV4_DST
	XMT_OFB_TCP_SRC
	XMT_OFB_TCP_DST
	XMT_OFB_UDP_SRC
	XMT_OFB_UDP_DST
	XMT_OFB_SCTP_SRC
	XMT_OFB_SCTP_DST
	XMT_OFB_ICMPV4_TYPE
	XMT_OFB_ICMPV4_CODE
	XMT_OFB_ARP_OP
	XMT_OFB_ARP_SPA
	XMT_OFB_ARP_TPA
	XMT_OFB_ARP_SHA
	XMT_OFB_ARP_THA
	XMT_OFB_IPV6_SRC
	XMT_OFB_IPV6_DST
	XMT_OFB_IPV6_FLABEL
	XMT_OFB_ICMPV6_TYPE
	XMT_OFB_ICMPV6_CODE
	XMT_OFB_IPV6_ND_TARGET
	XMT_OFB_IPV6_ND_SLL
	XMT_OFB_IPV6_ND_TLL
	XMT_OFB_MPLS_LABEL
	XMT_OFB_MPLS_TC
	XMT_OFP_MPLS_BOS
	XMT_OFB_PBB_ISID
	XMT_OFB_TUNNEL_ID
	XMT_OFB_IPV6_EXTHDR
)

type OXMField uint8

const (
	// Backward compatibility with NXM
	XMC_NXM_0 OXMClass = iota

	// Backward compatibility with NXM
	XMC_NXM_1 OXMClass = iota

	// The class XMC_OPENFLOW_BASIC contains
	// the basic set of OpenFlow match fields
	XMC_OPENFLOW_BASIC OXMClass = 0x8000

	// The optional class XMC_EXPERIMENTER is used
	// for experimenter matches
	XMC_EXPERIMENTER OXMClass = 0xffff
)

// OXM Class ID. The high order bit differentiate reserved
// classes from member classes. Classes 0x0000 to 0x7FFF are
// member classes, allocated by ONF. Classes 0x8000 to 0xFFFE
// are reserved classes, reserved for standardisation.
type OXMClass uint16

const (
	VID_NONE    VlanID = iota << 12
	VID_PRESENT VlanID = iota << 12
)

type VlanID uint16

const (
	IEH_NONEXT IPv6ExtHdrFlags = 1 << iota
	IEH_ESP    IPv6ExtHdrFlags = 1 << iota
	IEH_AUTH   IPv6ExtHdrFlags = 1 << iota
	IEH_DEST   IPv6ExtHdrFlags = 1 << iota
	IEH_FRAG   IPv6ExtHdrFlags = 1 << iota
	IEH_ROUTER IPv6ExtHdrFlags = 1 << iota
	IEH_HOP    IPv6ExtHdrFlags = 1 << iota
	IEH_UNREP  IPv6ExtHdrFlags = 1 << iota
	IEH_UNSEQ  IPv6ExtHdrFlags = 1 << iota
)

type IPv6ExtHdrFlags uint16

// Fields to match against flows
type Match struct {
	// Type indicates the match structure (set of fields that
	// compose the match) in use. The match type is placed in the
	// type field at the beginning of all match structures.
	Type      MatchType
	OXMFields []OXM
}

func (m *Match) ReadFrom(r io.Reader) (n int64, err error) {
	var length uint16
	n, err = binary.ReadSlice(r, binary.BigEndian, []interface{}{
		&m.Type, &length,
	})

	if err != nil {
		return
	}

	var nn int64

	buf := make([]byte, length)
	nn, err = binary.Read(r, binary.BigEndian, &buf)
	n += nn

	if err != nil {
		return
	}

	rbuf := bytes.NewBuffer(buf)
	for rbuf.Len() > 7 {
		var oxm OXM

		nn, err = oxm.ReadFrom(rbuf)
		n += nn

		if err != nil {
			return
		}

		m.OXMFields = append(m.OXMFields, oxm)
	}

	return
}

func (m *Match) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer

	for _, oxm := range m.OXMFields {
		_, err = oxm.WriteTo(&buf)
		if err != nil {
			return
		}
	}

	length := buf.Len() + 4

	_, err = buf.Write(make([]byte, 8-length%8))
	if err != nil {
		return
	}

	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		m.Type, uint16(length), buf.Bytes(),
	})
}

// The flow match fields are described using the OpenFlow Extensible
// Match (OXM) format, which is a compact type-length-value (TLV) format.
type OXM struct {
	// Match class that contains related match type
	Class OXMClass
	// Class-specific value, identifying one of the
	// match types within the match class.
	Field OXMField
	Mask  []byte
	Value []byte
}

func (oxm *OXM) ReadFrom(r io.Reader) (n int64, err error) {
	var length uint8

	n, err = binary.ReadSlice(r, binary.BigEndian, []interface{}{
		&oxm.Class, &oxm.Field, &length,
	})

	if err != nil {
		return
	}

	hasmask := (oxm.Field & 1) == 1
	oxm.Field >>= 1

	var m int64

	if hasmask {
		length /= 2
		oxm.Mask = make([]byte, length)

		m, err = binary.Read(r, binary.BigEndian, &oxm.Mask)
		n += m

		if err != nil {
			return
		}
	}

	oxm.Value = make([]byte, length)
	m, err = binary.Read(r, binary.BigEndian, &oxm.Value)
	return n + m, err
}

func (oxm *OXM) WriteTo(w io.Writer) (int64, error) {
	field := oxm.Field | (OXMField(len(oxm.Mask)) & 1)

	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		oxm.Class, field, uint8(len(oxm.Mask) + len(oxm.Value)), oxm.Mask, oxm.Value,
	})
}

type OXMExperimenterHeader struct {
	OXM          OXM
	Experimenter uint32
}

const (
	// Setup the next table in the lookup pipeline
	IT_GOTO_TABLE InstructionType = 1 + iota

	// Setup the metadata field for use later in pipeline
	IT_WRITE_METADATA InstructionType = 1 + iota

	// Write the action(s) onto the datapath action set
	IT_WRITE_ACTIONS InstructionType = 1 + iota

	// Applies the action(s) immediately
	IT_APPLY_ACTIONS InstructionType = 1 + iota

	// Clears all actions from the datapath action set
	IT_CLEAR_ACTIONS InstructionType = 1 + iota

	// Apply meter (rate limiter)
	IT_METER InstructionType = 1 + iota

	// Experimenter instruction
	IT_EXPERIMENTER InstructionType = 0xffff
)

type InstructionType uint16

// Instruction header that is common to all instructions. The length includes
// the header and any padding used to make the instruction 64-bit aligned.
// NB: The length of an instruction *must* always be a multiple of eight
type instructionhdr struct {
	// Instruction type
	Type InstructionType
	// Length of this struct in bytes
	Len uint16
}

type instruction interface {
	io.WriterTo
}

type Instructions []instruction

func (i Instructions) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer

	for _, inst := range i {
		_, err = inst.WriteTo(&buf)
		if err != nil {
			return
		}
	}

	return binary.Write(w, binary.BigEndian, buf.Bytes())
}

// Instruction structure for IT_GOTO_TABLE
type InstructionGotoTable struct {
	// TableID indicates the next table in the packet processing pipeline.
	TableID uint8
}

func (i InstructionGotoTable) WriteTo(w io.Writer) (int64, error) {
	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		instructionhdr{IT_GOTO_TABLE, 8}, i.TableID, pad3{},
	})
}

// Instruction structure for IT_WRITE_METADATA
type InstructionWriteMetadata struct {
	// Metadata value to write
	Metadata uint64
	// Metadata write bitmask
	MetadataMask uint64
}

func (i InstructionWriteMetadata) WriteTo(w io.Writer) (int64, error) {
	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		instructionhdr{IT_WRITE_METADATA, 24}, pad4{}, i.Metadata, i.MetadataMask,
	})
}

// Instruction structure for IT_WRITE/APPLY/CLEAR_ACTIONS
type InstructionActions struct {
	Type InstructionType
	// Actions associated with IT_WRITE_ACTIONS and IT_APPLY_ACTIONS.
	Actions Actions
	// For the Apply-Actions instruction, the actions field
	// is treated as a list and the actions are applied to
	// the packet in-order. For the Write-Actions instruction,
	// the actions field is treated as a set and the
	// actions are merged into the current action set.
	// For the Clear-Actions instruction, the structure does not contain any actions.
}

func (i InstructionActions) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer

	_, err = i.Actions.WriteTo(&buf)
	if err != nil {
		return
	}

	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		instructionhdr{i.Type, uint16(buf.Len()) + 8}, pad4{}, buf.Bytes(),
	})
}

// Instruction structure for IT_METER
type InstructionMeter struct {
	// MeterId indicates which meter to apply on the packet.
	MeterID uint32
}

func (i *InstructionMeter) WriteTo(w io.Writer) (int64, error) {
	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		instructionhdr{IT_METER, 8}, i.MeterID,
	})
}

const (
	// New flow
	FC_ADD FlowModCommand = iota

	// Modify all matching flows
	FC_MODIFY

	// Modify entry strictly matching
	// wildcards and priority
	FC_MODIFY_STRICT

	// Delete all matching flows
	FC_DELETE

	// Delete entry strictly matching
	// wildcards and priority
	FC_DELETE_STRICT
)

type FlowModCommand uint8

const (
	// When the FF_SEND_FLOW_REM flag is set, the switch must
	// send a flow removed message when the flow entry expires or is deleted.
	FF_SEND_FLOW_REM FlowModFlags = 1 << iota

	// When the FF_CHECK_OVERLAP flag is set, the switch must
	// check that there are no conflicting entries with the same
	// priority prior to inserting it in the flow table. If there is one,
	// the flow mod fails and an error message is returned
	FF_CHECK_OVERLAP FlowModFlags = 1 << iota

	// Reset flow packet and byte counts
	FF_RESET_COUNTS FlowModFlags = 1 << iota

	// When the FF_NO_PKT_COUNTS flag is set, the switch
	// does not need to keep track of the flow packet count
	FF_NO_PKT_COUNTS FlowModFlags = 1 << iota

	// When the FF_NO_BYT_COUNTS flag is set, the switch
	// does not need to keep track of the flow byte count.
	FF_NO_BYT_COUNTS FlowModFlags = 1 << iota
)

type FlowModFlags uint16

// Flow setup and teardown (controller -> datapath)
type FlowMod struct {
	// The cookie field is an opaque data value chosen by the
	// controller. This value appears in flow removed messages
	// and flow statistics, and can also be used to filter flow
	// statistics, flow modification and flow deletion
	Cookie uint64

	// If the CookieMast field is non-zero, it is used with the
	// cookie field to restrict flow matching while modifying or
	// deleting flow entries. This field is ignored by FC_ADD messages.
	// A value of 0 indicates no restriction
	CookieMask uint64

	// ID of the table to put the flow in. For FC_DELETE_* commands, TT_ALL
	// can also be used to delete matching flows from all tables.
	TableID Table

	// One of FlowModCommand
	Command FlowModCommand

	// Idle time before discarding (seconds). If the IdleTimeout is set
	// and the HardTimeout is zero, the entry must expire after IdleTimeout
	// seconds with no received traffic. If the IdleTimeout is zero and
	// the HardTimeout is set, the entry must expire in HardTimeout seconds
	// regardless of whether or not packets are hitting the entry.
	IdleTimeout uint16

	// Max time before discarding (seconds). If both IdleTimeout and
	// HardTimeout are set, the flow entry will timeout after IdleTimeout
	// seconds with no traffic, or HardTimeout seconds, whichever comes first.
	// If both IdleTimeout and HardTimeout are zero, the entry is considered
	// permanent and will never time out.
	HardTimeout uint16

	// Priority level of flow entry. The priority indicates priority within
	// the specified flow table table. Higher numbers indicate higher
	// priorities. This field is used only for FC_ADD messages when matching
	// and adding flow entries, and for FC_MODIFY_STRICT or FC_DELETE_STRICT
	// messages when matching flow entries.
	Priority uint16

	// The buffer_id refers to a packet buffered at the switch and sent
	// to the controller by a packet-in message. If no buffered packet
	// is associated with the flow mod, it must be set to NO_BUFFER.
	// A flow mod that includes a valid BufferID is effectively equivalent
	// to sending a two-message sequence of a flow mod and a packet-out to
	// P_TABLE, with the requirement that the switch must fully process
	// the flow mod before the packet out.
	BufferID uint32

	// For FC_DELETE* commands, require matching entries to
	// include this as an output port. A value of PP_ANY
	// indicates no restriction
	OutPort PortNo

	// For FC_DELETE* commands, require matching entries to
	// include this as an output group. A value of PG_ANY
	// indicates no restriction.
	OutGroup Group

	// One of FlowModFlags
	Flags FlowModFlags

	// Fields to match
	Match Match

	// The instructions field contains the instruction set
	// for the flow entry when adding or modifying entries.
	// If the instruction set is not valid or supported,
	// the switch must generate an error
	Instructions Instructions
}

func (f *FlowMod) Bytes() []byte {
	return Bytes(f)
}

func (f *FlowMod) WriteTo(w io.Writer) (n int64, err error) {
	var buf bytes.Buffer

	_, err = f.Match.WriteTo(&buf)
	if err != nil {
		return
	}

	_, err = f.Instructions.WriteTo(&buf)
	if err != nil {
		return
	}

	return binary.WriteSlice(w, binary.BigEndian, []interface{}{
		f.Cookie,
		f.CookieMask,
		f.TableID,
		f.Command,
		f.IdleTimeout,
		f.HardTimeout,
		f.Priority,
		f.BufferID,
		f.OutPort,
		f.OutGroup,
		f.Flags,
		pad2{},
		buf.Bytes(),
	})
}

const (
	RR_IDLE_TIMEOUT FlowRemovedReason = iota
	RR_HARD_TIMEOUT
	RR_DELETE
	RR_GROUP_DELETE
)

type FlowRemovedReason uint8

type FlowRemoved struct {
	Cookie       uint64
	Priority     uint16
	Reason       FlowRemovedReason
	TableID      Table
	DurationSec  uint32
	DurationNSec uint32
	IdleTimeout  uint16
	HardTimeout  uint16
	PacketCount  uint64
	ByteCount    uint64
	Match        Match
}

type FlowStatsRequest struct {
	TableID    Table
	_          pad3
	OutPort    PortNo
	OutGroup   Group
	_          pad4
	Cookie     uint64
	CookieMask uint64
	Match      Match
}

type FlowStats struct {
	Length       uint16
	TableID      Table
	_            pad1
	DurationSec  uint32
	DurationNSec uint32

	Priority    uint16
	IdleTimeout uint16
	HardTimeout uint16
	Flags       FlowModFlags
	_           pad4
	Cookie      uint64
	PacketCount uint64
	ByteCount   uint64
	Match       Match
}