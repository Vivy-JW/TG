using System;
using System.Xml.Linq;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Model
{
	// Token: 0x0200002A RID: 42
	public class SectionModel
	{
		// Token: 0x17000048 RID: 72
		// (get) Token: 0x060000F5 RID: 245 RVA: 0x000056C0 File Offset: 0x000038C0
		// (set) Token: 0x060000F6 RID: 246 RVA: 0x000056C8 File Offset: 0x000038C8
		public XElement Source { get; private set; }

		// Token: 0x17000049 RID: 73
		// (get) Token: 0x060000F7 RID: 247 RVA: 0x000056D1 File Offset: 0x000038D1
		public string Name
		{
			get
			{
				return this.Source.Attribute("id").Value;
			}
		}

		// Token: 0x1700004A RID: 74
		// (get) Token: 0x060000F8 RID: 248 RVA: 0x000056ED File Offset: 0x000038ED
		// (set) Token: 0x060000F9 RID: 249 RVA: 0x000056F5 File Offset: 0x000038F5
		public PackageKey ProgramKey { get; private set; }

		// Token: 0x1700004B RID: 75
		// (get) Token: 0x060000FA RID: 250 RVA: 0x000056FE File Offset: 0x000038FE
		public bool IsCustomized
		{
			get
			{
				return this.Status.Equals("u", StringComparison.CurrentCultureIgnoreCase) || !this.Source.Attribute("src").Value.Equals("s", StringComparison.CurrentCultureIgnoreCase);
			}
		}

		// Token: 0x1700004C RID: 76
		// (get) Token: 0x060000FB RID: 251 RVA: 0x0000573D File Offset: 0x0000393D
		// (set) Token: 0x060000FC RID: 252 RVA: 0x0000574C File Offset: 0x0000394C
		public string Content
		{
			get
			{
				return this.Source.Value;
			}
			set
			{
				if (!this.IsEditable)
				{
					throw new InvalidOperationException("Section is ReadOnly");
				}
				if (!string.Equals(this.Source.Value.ToString().Replace("\n\a", "").Replace("\a\n", "")
					.Replace("\a", ""), value.ToString().Replace("\n\a", "").Replace("\a\n", "")
					.Replace("\a", "")))
				{
					this.Status = "u";
				}
				this.Source.ReplaceNodes(new XCData(value));
			}
		}

		// Token: 0x1700004D RID: 77
		// (get) Token: 0x060000FD RID: 253 RVA: 0x00005801 File Offset: 0x00003A01
		// (set) Token: 0x060000FE RID: 254 RVA: 0x00005820 File Offset: 0x00003A20
		public string Status
		{
			get
			{
				return this.Source.Attribute("status").Value;
			}
			private set
			{
				bool isTopstdMode = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode;
				this.Source.SetAttributeValue("src", isTopstdMode ? "s" : ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ENV);
				this.Source.SetAttributeValue("modi_by_topstd", isTopstdMode ? "Y" : "");
				this.Source.Attribute("status").Value = value;
			}
		}

		// Token: 0x1700004E RID: 78
		// (get) Token: 0x060000FF RID: 255 RVA: 0x000058BC File Offset: 0x00003ABC
		public bool IsModified
		{
			get
			{
				string text = ((this.Source.Attribute("ch") == null) ? string.Empty : this.Source.Attribute("ch").Value);
				return string.Equals("u", this.Status) || string.Equals("Y", text, StringComparison.CurrentCultureIgnoreCase);
			}
		}

		// Token: 0x1700004F RID: 79
		// (get) Token: 0x06000100 RID: 256 RVA: 0x00005922 File Offset: 0x00003B22
		// (set) Token: 0x06000101 RID: 257 RVA: 0x0000592A File Offset: 0x00003B2A
		public bool IsReadOnly { get; set; }

		// Token: 0x17000050 RID: 80
		// (get) Token: 0x06000102 RID: 258 RVA: 0x00005934 File Offset: 0x00003B34
		public bool IsEditable
		{
			get
			{
				if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).Type == "G" && ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsSectionModify)
				{
					string text = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ProgramKey.Program + ".other_function";
					string text2 = ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).ProgramKey.Program + ".other_report";
					return this.Name != text && this.Name != text2;
				}
				if (this.IsReadOnly)
				{
					return false;
				}
				if (this.Source.Attribute("readonly") != null && this.Source.Attribute("readonly").Value.Equals("Y", StringComparison.CurrentCultureIgnoreCase))
				{
					return false;
				}
				string text3;
				if (ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).IsTopstdMode && (text3 = this.SRC.ToLower()) != null)
				{
					if (text3 == "c")
					{
						return false;
					}
					if (text3 == "s" || text3 == "m")
					{
						return this.Source.Attribute("modi_by_topstd") == null || ("Y" == this.Source.Attribute("modi_by_topstd").Value && "u" == this.Status) || "" == this.Status;
					}
				}
				return !("topstd" == ResourceController.GetInstance().GetProgramInfo(this.ProgramKey).login_user) || !(this.SRC.ToLower() == "c");
			}
		}

		// Token: 0x17000051 RID: 81
		// (get) Token: 0x06000103 RID: 259 RVA: 0x00005B2B File Offset: 0x00003D2B
		// (set) Token: 0x06000104 RID: 260 RVA: 0x00005B33 File Offset: 0x00003D33
		public Guid ID { get; private set; }

		// Token: 0x17000052 RID: 82
		// (get) Token: 0x06000105 RID: 261 RVA: 0x00005B3C File Offset: 0x00003D3C
		// (set) Token: 0x06000106 RID: 262 RVA: 0x00005B44 File Offset: 0x00003D44
		public string SRC { get; private set; }

		// Token: 0x06000107 RID: 263 RVA: 0x00005B4D File Offset: 0x00003D4D
		public SectionModel(PackageKey key, XElement ele)
		{
			this.ProgramKey = key;
			this.ID = Guid.NewGuid();
			this.Source = ele;
			this.SRC = ele.Attribute("src").Value;
		}

		// Token: 0x06000108 RID: 264 RVA: 0x00005B8C File Offset: 0x00003D8C
		public XElement ToXElement()
		{
			if (SettingManager.Get().GetTzpManger(this.ProgramKey).IsDiff)
			{
				this.Source.ReplaceNodes(new XCData(this.Source.Value.Replace("\n\a", "").Replace("\a", "")));
			}
			return this.Source;
		}

		// Token: 0x06000109 RID: 265 RVA: 0x00005BF0 File Offset: 0x00003DF0
		public static SectionModel Clone(PackageKey key, string source, string newName)
		{
			SectionModel sectionModel = new SectionModel(key, XElement.Parse(source));
			sectionModel.Source.Attribute("id").Value = newName;
			sectionModel.Source.Attribute("status").Value = "u";
			sectionModel.Source.Attribute("ver").Value = ResourceController.GetInstance().GetProgramInfo(key).Ver;
			sectionModel.Source.ReplaceNodes(new XCData(Environment.NewLine));
			return sectionModel;
		}
	}
}
