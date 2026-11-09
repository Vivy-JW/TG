using System;

namespace SpecDesignerCommon.Exceptions
{
	// Token: 0x02000058 RID: 88
	public class VersionIncompatibleException : Exception
	{
		// Token: 0x060002F4 RID: 756 RVA: 0x0000C945 File Offset: 0x0000AB45
		public VersionIncompatibleException(Version _current, Version _remote)
		{
			this.Current = _current;
			this.Remote = _remote;
		}

		// Token: 0x060002F5 RID: 757 RVA: 0x0000C95B File Offset: 0x0000AB5B
		public VersionIncompatibleException(string message)
			: base(message)
		{
		}

		// Token: 0x04000117 RID: 279
		public Version Current;

		// Token: 0x04000118 RID: 280
		public Version Remote;
	}
}
