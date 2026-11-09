using System;

namespace SpecDesignerCommon.Events
{
	// Token: 0x020000A0 RID: 160
	public class SearchKeywordEventArgs
	{
		// Token: 0x06000665 RID: 1637 RVA: 0x0001C960 File Offset: 0x0001AB60
		public SearchKeywordEventArgs(PackageKey key, string keyword, DiffType type)
		{
			this.ProgramKey = key;
			this.Keyword = keyword;
			this.IsMatchCase = true;
			this.EditableAreaOnly = false;
			this.SelectedOnly = false;
			this.IsSearchSpec = false;
			this.IsLoop = false;
			this.IsRegExMode = true;
			this.Type = type;
		}

		// Token: 0x06000666 RID: 1638 RVA: 0x0001C9B2 File Offset: 0x0001ABB2
		public SearchKeywordEventArgs(PackageKey key, string keyword, bool isMatchCase, bool editableAreaOnly, bool selectedOnly, bool isRegExMode, bool isSearchSpec)
		{
			this.ProgramKey = key;
			this.Keyword = keyword;
			this.IsMatchCase = isMatchCase;
			this.EditableAreaOnly = editableAreaOnly;
			this.SelectedOnly = selectedOnly;
			this.IsSearchSpec = isSearchSpec;
			this.IsLoop = false;
		}

		// Token: 0x06000667 RID: 1639 RVA: 0x0001C9EE File Offset: 0x0001ABEE
		public SearchKeywordEventArgs(PackageKey key, string keyword, bool isMatchCase, bool editableAreaOnly, bool selectedOnly)
			: this(key, keyword, isMatchCase, editableAreaOnly, selectedOnly, false, false)
		{
		}

		// Token: 0x06000668 RID: 1640 RVA: 0x0001C9FF File Offset: 0x0001ABFF
		public SearchKeywordEventArgs(PackageKey key, string keyword, bool isMatchCase, bool editableAreaOnly)
			: this(key, keyword, isMatchCase, editableAreaOnly, false)
		{
		}

		// Token: 0x06000669 RID: 1641 RVA: 0x0001CA0D File Offset: 0x0001AC0D
		public SearchKeywordEventArgs(PackageKey key, string keyword, bool isMatchCase)
			: this(key, keyword, isMatchCase, false)
		{
		}

		// Token: 0x0600066A RID: 1642 RVA: 0x0001CA19 File Offset: 0x0001AC19
		public SearchKeywordEventArgs(PackageKey key, string keyword)
			: this(key, keyword, false)
		{
		}

		// Token: 0x0600066B RID: 1643 RVA: 0x0001CA24 File Offset: 0x0001AC24
		public SearchKeywordEventArgs()
		{
			this.IsSearchFromCaret = false;
		}

		// Token: 0x170001D1 RID: 465
		// (get) Token: 0x0600066C RID: 1644 RVA: 0x0001CA33 File Offset: 0x0001AC33
		// (set) Token: 0x0600066D RID: 1645 RVA: 0x0001CA3B File Offset: 0x0001AC3B
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x170001D2 RID: 466
		// (get) Token: 0x0600066E RID: 1646 RVA: 0x0001CA44 File Offset: 0x0001AC44
		// (set) Token: 0x0600066F RID: 1647 RVA: 0x0001CA4C File Offset: 0x0001AC4C
		public string Keyword { get; private set; }

		// Token: 0x170001D3 RID: 467
		// (get) Token: 0x06000670 RID: 1648 RVA: 0x0001CA55 File Offset: 0x0001AC55
		// (set) Token: 0x06000671 RID: 1649 RVA: 0x0001CA5D File Offset: 0x0001AC5D
		public bool IsMatchCase { get; private set; }

		// Token: 0x170001D4 RID: 468
		// (get) Token: 0x06000672 RID: 1650 RVA: 0x0001CA66 File Offset: 0x0001AC66
		// (set) Token: 0x06000673 RID: 1651 RVA: 0x0001CA6E File Offset: 0x0001AC6E
		public bool IsRegExMode { get; set; }

		// Token: 0x170001D5 RID: 469
		// (get) Token: 0x06000674 RID: 1652 RVA: 0x0001CA77 File Offset: 0x0001AC77
		// (set) Token: 0x06000675 RID: 1653 RVA: 0x0001CA7F File Offset: 0x0001AC7F
		public bool EditableAreaOnly { get; private set; }

		// Token: 0x170001D6 RID: 470
		// (get) Token: 0x06000676 RID: 1654 RVA: 0x0001CA88 File Offset: 0x0001AC88
		// (set) Token: 0x06000677 RID: 1655 RVA: 0x0001CA90 File Offset: 0x0001AC90
		public bool SelectedOnly { get; private set; }

		// Token: 0x170001D7 RID: 471
		// (get) Token: 0x06000678 RID: 1656 RVA: 0x0001CA99 File Offset: 0x0001AC99
		// (set) Token: 0x06000679 RID: 1657 RVA: 0x0001CAA1 File Offset: 0x0001ACA1
		public string ReplaceAs { get; set; }

		// Token: 0x170001D8 RID: 472
		// (get) Token: 0x0600067A RID: 1658 RVA: 0x0001CAAA File Offset: 0x0001ACAA
		// (set) Token: 0x0600067B RID: 1659 RVA: 0x0001CAB2 File Offset: 0x0001ACB2
		public bool IsSearchSpec { get; set; }

		// Token: 0x170001D9 RID: 473
		// (get) Token: 0x0600067C RID: 1660 RVA: 0x0001CABB File Offset: 0x0001ACBB
		// (set) Token: 0x0600067D RID: 1661 RVA: 0x0001CAC3 File Offset: 0x0001ACC3
		public bool IsSearchFromCaret { get; set; }

		// Token: 0x170001DA RID: 474
		// (get) Token: 0x0600067E RID: 1662 RVA: 0x0001CACC File Offset: 0x0001ACCC
		// (set) Token: 0x0600067F RID: 1663 RVA: 0x0001CAD4 File Offset: 0x0001ACD4
		public SearchDirection Direction { get; set; }

		// Token: 0x170001DB RID: 475
		// (get) Token: 0x06000680 RID: 1664 RVA: 0x0001CADD File Offset: 0x0001ACDD
		// (set) Token: 0x06000681 RID: 1665 RVA: 0x0001CAE5 File Offset: 0x0001ACE5
		public bool IsLoop { get; set; }

		// Token: 0x170001DC RID: 476
		// (get) Token: 0x06000682 RID: 1666 RVA: 0x0001CAEE File Offset: 0x0001ACEE
		// (set) Token: 0x06000683 RID: 1667 RVA: 0x0001CAF6 File Offset: 0x0001ACF6
		public DiffType Type { get; set; }
	}
}
