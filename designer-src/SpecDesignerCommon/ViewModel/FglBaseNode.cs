using System;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x02000044 RID: 68
	public abstract class FglBaseNode
	{
		// Token: 0x17000086 RID: 134
		// (get) Token: 0x0600022C RID: 556 RVA: 0x000099C2 File Offset: 0x00007BC2
		// (set) Token: 0x0600022D RID: 557 RVA: 0x000099CA File Offset: 0x00007BCA
		public CodeSpecStatus Status
		{
			get
			{
				return this._status;
			}
			set
			{
				this._status = value;
				if ((value & CodeSpecStatus.MODIFY) == CodeSpecStatus.MODIFY || (value & CodeSpecStatus.DELETE) == CodeSpecStatus.DELETE || (value & CodeSpecStatus.CREATE) == CodeSpecStatus.CREATE)
				{
					this.Parent.Status = CodeSpecStatus.MODIFY | this.Parent.Status;
				}
			}
		}

		// Token: 0x17000087 RID: 135
		// (get) Token: 0x0600022E RID: 558 RVA: 0x000099FD File Offset: 0x00007BFD
		public PackageKey ProgramKey
		{
			get
			{
				return this.Parent.ProgramKey;
			}
		}

		// Token: 0x040000CF RID: 207
		private CodeSpecStatus _status;

		// Token: 0x040000D0 RID: 208
		protected FglComponent Parent;
	}
}
