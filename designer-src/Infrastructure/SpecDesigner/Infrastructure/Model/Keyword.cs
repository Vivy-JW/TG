using System;
using System.Windows;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x02000049 RID: 73
	public class Keyword : IIntellisenseModel
	{
		// Token: 0x170000C4 RID: 196
		// (get) Token: 0x0600023A RID: 570 RVA: 0x0000B0F0 File Offset: 0x000092F0
		// (set) Token: 0x0600023B RID: 571 RVA: 0x0000B0F8 File Offset: 0x000092F8
		public bool IsFavorited { get; set; }

		// Token: 0x170000C5 RID: 197
		// (get) Token: 0x0600023C RID: 572 RVA: 0x0000B101 File Offset: 0x00009301
		public string FullName
		{
			get
			{
				return this.Name;
			}
		}

		// Token: 0x170000C6 RID: 198
		// (get) Token: 0x0600023D RID: 573 RVA: 0x0000B109 File Offset: 0x00009309
		// (set) Token: 0x0600023E RID: 574 RVA: 0x0000B111 File Offset: 0x00009311
		public string Name { get; set; }

		// Token: 0x170000C7 RID: 199
		// (get) Token: 0x0600023F RID: 575 RVA: 0x0000B11A File Offset: 0x0000931A
		public IntellisenseEnum Type
		{
			get
			{
				return IntellisenseEnum.Keyword;
			}
		}

		// Token: 0x170000C8 RID: 200
		// (get) Token: 0x06000240 RID: 576 RVA: 0x0000B11D File Offset: 0x0000931D
		// (set) Token: 0x06000241 RID: 577 RVA: 0x0000B125 File Offset: 0x00009325
		public string Description
		{
			get
			{
				return this._description;
			}
			set
			{
				this._description = value.Trim();
			}
		}

		// Token: 0x06000242 RID: 578 RVA: 0x0000B133 File Offset: 0x00009333
		public Keyword(string name)
		{
			this.Name = name;
			this.IsFavorited = false;
			this._description = Application.Current.FindResource("CE_GeneroKey") as string;
		}

		// Token: 0x06000243 RID: 579 RVA: 0x0000B163 File Offset: 0x00009363
		public override string ToString()
		{
			return this.Description;
		}

		// Token: 0x040000E6 RID: 230
		private string _description;
	}
}
