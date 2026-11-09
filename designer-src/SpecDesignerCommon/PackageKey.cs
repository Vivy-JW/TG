using System;
using System.Collections.Generic;

namespace SpecDesignerCommon
{
	// Token: 0x0200000C RID: 12
	[Serializable]
	public class PackageKey
	{
		// Token: 0x0600005E RID: 94 RVA: 0x00003850 File Offset: 0x00001A50
		public PackageKey(string p, TzpType t)
		{
			this.Program = p;
			this.PackType = t;
		}

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x0600005F RID: 95 RVA: 0x00003866 File Offset: 0x00001A66
		// (set) Token: 0x06000060 RID: 96 RVA: 0x0000386E File Offset: 0x00001A6E
		public string Program { get; private set; }

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000061 RID: 97 RVA: 0x00003877 File Offset: 0x00001A77
		// (set) Token: 0x06000062 RID: 98 RVA: 0x0000387F File Offset: 0x00001A7F
		public TzpType PackType { get; private set; }

		// Token: 0x17000012 RID: 18
		// (get) Token: 0x06000063 RID: 99 RVA: 0x00003888 File Offset: 0x00001A88
		// (set) Token: 0x06000064 RID: 100 RVA: 0x00003890 File Offset: 0x00001A90
		public MemoType Memo
		{
			get
			{
				return this._memo;
			}
			set
			{
				this._memo = value;
			}
		}

		// Token: 0x06000065 RID: 101 RVA: 0x0000389C File Offset: 0x00001A9C
		public override bool Equals(object obj)
		{
			if (!(obj is PackageKey))
			{
				return false;
			}
			PackageKey packageKey = (PackageKey)obj;
			return this.Program == packageKey.Program && this.PackType == packageKey.PackType && this.Memo == packageKey.Memo;
		}

		// Token: 0x06000066 RID: 102 RVA: 0x000038EB File Offset: 0x00001AEB
		public static bool operator ==(PackageKey x, PackageKey y)
		{
			if (object.ReferenceEquals(x, null))
			{
				return object.ReferenceEquals(y, null);
			}
			return x.Equals(y);
		}

		// Token: 0x06000067 RID: 103 RVA: 0x00003905 File Offset: 0x00001B05
		public static bool operator !=(PackageKey x, PackageKey y)
		{
			return !(x == y);
		}

		// Token: 0x04000024 RID: 36
		private MemoType _memo;

		// Token: 0x0200000D RID: 13
		public class PackageKeyComparer : IEqualityComparer<PackageKey>
		{
			// Token: 0x06000068 RID: 104 RVA: 0x00003911 File Offset: 0x00001B11
			public bool Equals(PackageKey x, PackageKey y)
			{
				return x.PackType == y.PackType && x.Program == y.Program && x.Memo == y.Memo;
			}

			// Token: 0x06000069 RID: 105 RVA: 0x00003944 File Offset: 0x00001B44
			public int GetHashCode(PackageKey obj)
			{
				return obj.PackType.GetHashCode() ^ obj.Program.GetHashCode() ^ obj.Memo.GetHashCode();
			}
		}
	}
}
