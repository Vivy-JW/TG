using System;
using SpecDesignerCommon;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000037 RID: 55
	public class ItemData
	{
		// Token: 0x06000174 RID: 372 RVA: 0x0000AFA8 File Offset: 0x000091A8
		public ItemData(string widget, string componentName, int index)
		{
			widget = widget.ToLower();
			string text;
			if ((text = widget) != null)
			{
				if (text == "combobox")
				{
					this._text = string.Format("cbo_{0}.{1}", componentName, index);
					return;
				}
				if (text == "radiogroup")
				{
					this._text = string.Format("rdo_{0}.{1}", componentName, index);
					return;
				}
			}
			this._text = string.Format("cbo_{0}.{1}", componentName, index);
		}

		// Token: 0x17000027 RID: 39
		// (get) Token: 0x06000175 RID: 373 RVA: 0x0000B04A File Offset: 0x0000924A
		// (set) Token: 0x06000176 RID: 374 RVA: 0x0000B052 File Offset: 0x00009252
		public PackageKey Program
		{
			get
			{
				return this._program;
			}
			set
			{
				this._program = value;
				this.IsChanged = true;
			}
		}

		// Token: 0x17000028 RID: 40
		// (get) Token: 0x06000177 RID: 375 RVA: 0x0000B062 File Offset: 0x00009262
		// (set) Token: 0x06000178 RID: 376 RVA: 0x0000B06A File Offset: 0x0000926A
		public string Name
		{
			get
			{
				return this._name;
			}
			set
			{
				this._name = value;
				this.IsChanged = true;
			}
		}

		// Token: 0x17000029 RID: 41
		// (get) Token: 0x06000179 RID: 377 RVA: 0x0000B07A File Offset: 0x0000927A
		// (set) Token: 0x0600017A RID: 378 RVA: 0x0000B082 File Offset: 0x00009282
		public string Text
		{
			get
			{
				return this._text;
			}
			set
			{
				this._text = value;
				this.IsChanged = true;
			}
		}

		// Token: 0x1700002A RID: 42
		// (get) Token: 0x0600017B RID: 379 RVA: 0x0000B092 File Offset: 0x00009292
		// (set) Token: 0x0600017C RID: 380 RVA: 0x0000B0AD File Offset: 0x000092AD
		public string Description
		{
			get
			{
				if (this._description == null)
				{
					this._description = "";
				}
				return this._description;
			}
			set
			{
				this._description = value;
				this.IsChanged = true;
			}
		}

		// Token: 0x1700002B RID: 43
		// (get) Token: 0x0600017D RID: 381 RVA: 0x0000B0BD File Offset: 0x000092BD
		// (set) Token: 0x0600017E RID: 382 RVA: 0x0000B0C5 File Offset: 0x000092C5
		public bool IsChanged { get; set; }

		// Token: 0x040000B3 RID: 179
		private PackageKey _program;

		// Token: 0x040000B4 RID: 180
		private string _name = "";

		// Token: 0x040000B5 RID: 181
		private string _text = "";

		// Token: 0x040000B6 RID: 182
		private string _description = "";
	}
}
