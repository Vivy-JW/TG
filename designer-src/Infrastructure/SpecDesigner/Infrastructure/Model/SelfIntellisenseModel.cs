using System;
using System.ComponentModel;
using System.Windows;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200002E RID: 46
	public class SelfIntellisenseModel : IIntellisenseModel
	{
		// Token: 0x06000133 RID: 307 RVA: 0x000069BF File Offset: 0x00004BBF
		public SelfIntellisenseModel(string name)
		{
			this.Name = name;
		}

		// Token: 0x17000063 RID: 99
		// (get) Token: 0x06000134 RID: 308 RVA: 0x000069D5 File Offset: 0x00004BD5
		public string FullName
		{
			get
			{
				return this.Name;
			}
		}

		// Token: 0x17000064 RID: 100
		// (get) Token: 0x06000135 RID: 309 RVA: 0x000069DD File Offset: 0x00004BDD
		// (set) Token: 0x06000136 RID: 310 RVA: 0x000069E5 File Offset: 0x00004BE5
		public bool IsFavorited
		{
			get
			{
				return this._isFavorited;
			}
			set
			{
			}
		}

		// Token: 0x17000065 RID: 101
		// (get) Token: 0x06000137 RID: 311 RVA: 0x000069E7 File Offset: 0x00004BE7
		// (set) Token: 0x06000138 RID: 312 RVA: 0x000069EF File Offset: 0x00004BEF
		[DefaultValue("")]
		public string Name { get; set; }

		// Token: 0x17000066 RID: 102
		// (get) Token: 0x06000139 RID: 313 RVA: 0x000069F8 File Offset: 0x00004BF8
		// (set) Token: 0x0600013A RID: 314 RVA: 0x00006A00 File Offset: 0x00004C00
		[DefaultValue("")]
		public string Description { get; set; }

		// Token: 0x17000067 RID: 103
		// (get) Token: 0x0600013B RID: 315 RVA: 0x00006A09 File Offset: 0x00004C09
		public IntellisenseEnum Type
		{
			get
			{
				return IntellisenseEnum.Self;
			}
		}

		// Token: 0x17000068 RID: 104
		// (get) Token: 0x0600013C RID: 316 RVA: 0x00006A0C File Offset: 0x00004C0C
		public string Content
		{
			get
			{
				return null;
			}
		}

		// Token: 0x0600013D RID: 317 RVA: 0x00006A10 File Offset: 0x00004C10
		public static SelfIntellisenseModel Create(string name)
		{
			if (string.IsNullOrWhiteSpace(name))
			{
				return null;
			}
			return new SelfIntellisenseModel(name)
			{
				IsFavorited = true
			};
		}

		// Token: 0x0600013E RID: 318 RVA: 0x00006A38 File Offset: 0x00004C38
		public override string ToString()
		{
			return string.Format("{0}({1})", this.Name, Application.Current.FindResource("CE_SelfKey") as string);
		}

		// Token: 0x04000079 RID: 121
		private bool _isFavorited = true;
	}
}
