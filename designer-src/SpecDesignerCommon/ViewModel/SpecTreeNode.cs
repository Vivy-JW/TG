using System;
using System.ComponentModel;
using System.Linq;
using System.Xml.Linq;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.UndoRedo;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000A2 RID: 162
	public class SpecTreeNode : AbstractSpecNode, IDataErrorInfo
	{
		// Token: 0x060006A9 RID: 1705 RVA: 0x0001E13E File Offset: 0x0001C33E
		public SpecTreeNode(PackageKey key, XElement source)
			: base(key, source)
		{
		}

		// Token: 0x170001DD RID: 477
		// (get) Token: 0x060006AA RID: 1706 RVA: 0x0001E148 File Offset: 0x0001C348
		// (set) Token: 0x060006AB RID: 1707 RVA: 0x0001E155 File Offset: 0x0001C355
		public override string Name
		{
			get
			{
				return base.GetAttribute("name");
			}
			set
			{
				this.OnPropertyChanged("Name");
			}
		}

		// Token: 0x060006AC RID: 1708 RVA: 0x0001E162 File Offset: 0x0001C362
		internal override void SetName(FormSpecModel model, string newName)
		{
			this.OnPropertyChanged("Name");
		}

		// Token: 0x170001DE RID: 478
		// (get) Token: 0x060006AD RID: 1709 RVA: 0x0001E16F File Offset: 0x0001C36F
		public string Kind
		{
			get
			{
				return base.GetAttribute("kind");
			}
		}

		// Token: 0x170001DF RID: 479
		// (get) Token: 0x060006AE RID: 1710 RVA: 0x0001E17C File Offset: 0x0001C37C
		public string Att
		{
			get
			{
				return base.GetAttribute("att");
			}
		}

		// Token: 0x170001E0 RID: 480
		// (get) Token: 0x060006AF RID: 1711 RVA: 0x0001E189 File Offset: 0x0001C389
		// (set) Token: 0x060006B0 RID: 1712 RVA: 0x0001E196 File Offset: 0x0001C396
		public string Type
		{
			get
			{
				return this.GetType("");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				this.SetType("", value);
			}
		}

		// Token: 0x170001E1 RID: 481
		// (get) Token: 0x060006B1 RID: 1713 RVA: 0x0001E1AD File Offset: 0x0001C3AD
		// (set) Token: 0x060006B2 RID: 1714 RVA: 0x0001E1BA File Offset: 0x0001C3BA
		public string Type2
		{
			get
			{
				return this.GetType("2");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				this.SetType("2", value);
			}
		}

		// Token: 0x170001E2 RID: 482
		// (get) Token: 0x060006B3 RID: 1715 RVA: 0x0001E1D1 File Offset: 0x0001C3D1
		// (set) Token: 0x060006B4 RID: 1716 RVA: 0x0001E1DE File Offset: 0x0001C3DE
		public string Type3
		{
			get
			{
				return this.GetType("3");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				this.SetType("3", value);
			}
		}

		// Token: 0x170001E3 RID: 483
		// (get) Token: 0x060006B5 RID: 1717 RVA: 0x0001E1F5 File Offset: 0x0001C3F5
		// (set) Token: 0x060006B6 RID: 1718 RVA: 0x0001E202 File Offset: 0x0001C402
		public string Type4
		{
			get
			{
				return this.GetType("4");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				this.SetType("4", value);
			}
		}

		// Token: 0x170001E4 RID: 484
		// (get) Token: 0x060006B7 RID: 1719 RVA: 0x0001E219 File Offset: 0x0001C419
		// (set) Token: 0x060006B8 RID: 1720 RVA: 0x0001E226 File Offset: 0x0001C426
		public string Type5
		{
			get
			{
				return this.GetType("5");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				this.SetType("5", value);
			}
		}

		// Token: 0x170001E5 RID: 485
		// (get) Token: 0x060006B9 RID: 1721 RVA: 0x0001E23D File Offset: 0x0001C43D
		// (set) Token: 0x060006BA RID: 1722 RVA: 0x0001E24A File Offset: 0x0001C44A
		public string Type6
		{
			get
			{
				return this.GetType("6");
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				this.SetType("6", value);
			}
		}

		// Token: 0x060006BB RID: 1723 RVA: 0x0001E264 File Offset: 0x0001C464
		private string GetType(string seq)
		{
			XElement xelement = base.Source.Element("type" + seq);
			string text = ".";
			if (xelement.Attribute("table").Value == "" || xelement.Attribute("col").Value == "")
			{
				text = "";
			}
			return string.Format("{0}{1}{2}", xelement.Attribute("table").Value, text, xelement.Attribute("col").Value);
		}

		// Token: 0x060006BC RID: 1724 RVA: 0x0001E310 File Offset: 0x0001C510
		private void SetType(string seq, string value)
		{
			string[] array = value.Split(new char[] { '.' });
			string text = "";
			string text2 = value;
			if (array.Count<string>() == 2)
			{
				text = array[0];
				text2 = array[1];
			}
			SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, string.Format("type{0}", seq), string.Format("Type{0}", seq));
			specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", text);
			specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", text2);
			SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
		}

		// Token: 0x170001E6 RID: 486
		// (get) Token: 0x060006BD RID: 1725 RVA: 0x0001E398 File Offset: 0x0001C598
		// (set) Token: 0x060006BE RID: 1726 RVA: 0x0001E440 File Offset: 0x0001C640
		public string Id
		{
			get
			{
				XElement xelement = base.Source.Element("id");
				string text = ".";
				if (xelement.Attribute("table").Value == "" || xelement.Attribute("col").Value == "")
				{
					text = "";
				}
				return string.Format("{0}{1}{2}", xelement.Attribute("table").Value, text, xelement.Attribute("col").Value);
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				string[] array = value.Split(new char[] { '.' });
				string text = "";
				string text2 = value;
				if (array.Count<string>() == 2)
				{
					text = array[0];
					text2 = array[1];
				}
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "id", "Id");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", text);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", text2);
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001E7 RID: 487
		// (get) Token: 0x060006BF RID: 1727 RVA: 0x0001E4C8 File Offset: 0x0001C6C8
		// (set) Token: 0x060006C0 RID: 1728 RVA: 0x0001E570 File Offset: 0x0001C770
		public string Pid
		{
			get
			{
				XElement xelement = base.Source.Element("pid");
				string text = ".";
				if (xelement.Attribute("table").Value == "" || xelement.Attribute("col").Value == "")
				{
					text = "";
				}
				return string.Format("{0}{1}{2}", xelement.Attribute("table").Value, text, xelement.Attribute("col").Value);
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				string[] array = value.Split(new char[] { '.' });
				string text = "";
				string text2 = value;
				if (array.Count<string>() == 2)
				{
					text = array[0];
					text2 = array[1];
				}
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "pid", "Pid");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", text);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", text2);
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001E8 RID: 488
		// (get) Token: 0x060006C1 RID: 1729 RVA: 0x0001E5F8 File Offset: 0x0001C7F8
		// (set) Token: 0x060006C2 RID: 1730 RVA: 0x0001E6A0 File Offset: 0x0001C8A0
		public string Desc
		{
			get
			{
				XElement xelement = base.Source.Element("desc");
				string text = ".";
				if (xelement.Attribute("table").Value == "" || xelement.Attribute("col").Value == "")
				{
					text = "";
				}
				return string.Format("{0}{1}{2}", xelement.Attribute("table").Value, text, xelement.Attribute("col").Value);
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				string[] array = value.Split(new char[] { '.' });
				string text = "";
				string text2 = value;
				if (array.Count<string>() == 2)
				{
					text = array[0];
					text2 = array[1];
				}
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "desc", "Desc");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", text);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", text2);
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001E9 RID: 489
		// (get) Token: 0x060006C3 RID: 1731 RVA: 0x0001E728 File Offset: 0x0001C928
		// (set) Token: 0x060006C4 RID: 1732 RVA: 0x0001E760 File Offset: 0x0001C960
		public string Speed
		{
			get
			{
				XElement xelement = base.Source.Element("speed");
				return xelement.Attribute("table").Value;
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "speed", "Speed");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", value);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", "");
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001EA RID: 490
		// (get) Token: 0x060006C5 RID: 1733 RVA: 0x0001E7BC File Offset: 0x0001C9BC
		// (set) Token: 0x060006C6 RID: 1734 RVA: 0x0001E7F4 File Offset: 0x0001C9F4
		public string Stype
		{
			get
			{
				XElement xelement = base.Source.Element("stype");
				return xelement.Attribute("col").Value;
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				string speed = this.Speed;
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "stype", "Stype");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", speed);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", value);
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001EB RID: 491
		// (get) Token: 0x060006C7 RID: 1735 RVA: 0x0001E854 File Offset: 0x0001CA54
		// (set) Token: 0x060006C8 RID: 1736 RVA: 0x0001E88C File Offset: 0x0001CA8C
		public string Sid
		{
			get
			{
				XElement xelement = base.Source.Element("sid");
				return xelement.Attribute("col").Value;
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				string speed = this.Speed;
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "stype", "Stype");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", speed);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", value);
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001EC RID: 492
		// (get) Token: 0x060006C9 RID: 1737 RVA: 0x0001E8EC File Offset: 0x0001CAEC
		// (set) Token: 0x060006CA RID: 1738 RVA: 0x0001E924 File Offset: 0x0001CB24
		public string Spid
		{
			get
			{
				XElement xelement = base.Source.Element("spid");
				return xelement.Attribute("col").Value;
			}
			set
			{
				if (base.IsCited)
				{
					return;
				}
				string speed = this.Speed;
				SpecTreeAttributeUndoRedoCommand specTreeAttributeUndoRedoCommand = new SpecTreeAttributeUndoRedoCommand(this, "stype", "Stype");
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("table", speed);
				specTreeAttributeUndoRedoCommand.AddAttributeChanged("col", value);
				SettingManager.Get().GetUndoRedoManager(base.ProgramKey).AddThenExecute(specTreeAttributeUndoRedoCommand);
			}
		}

		// Token: 0x170001ED RID: 493
		// (get) Token: 0x060006CB RID: 1739 RVA: 0x0001E982 File Offset: 0x0001CB82
		// (set) Token: 0x060006CC RID: 1740 RVA: 0x0001E985 File Offset: 0x0001CB85
		public override string CDATA
		{
			get
			{
				return null;
			}
			set
			{
			}
		}

		// Token: 0x170001EE RID: 494
		// (get) Token: 0x060006CD RID: 1741 RVA: 0x0001E9E0 File Offset: 0x0001CBE0
		public override XElement CitedSpec
		{
			get
			{
				if (SettingManager.Get().GetTzpManger(base.ProgramKey).IsStandardProgram)
				{
					return null;
				}
				XElement citeSTD = SettingManager.Get().GetTzpManger(base.ProgramKey).SpecificationInfo.CiteSTD;
				if (citeSTD == null)
				{
					return null;
				}
				return (from e in citeSTD.Elements("tree")
					where e.Attribute("name").Value == this.Name && e.Attribute("status").Value != ReflectionHelpers.GetCustomDescription(SpecStatus.DELETE)
					select e).FirstOrDefault<XElement>();
			}
		}

		// Token: 0x060006CE RID: 1742 RVA: 0x0001EA50 File Offset: 0x0001CC50
		public static SpecTreeNode Create(SpecificationInfo info, string name)
		{
			XElement xelement = XElement.Parse("\r\n<tree ver='' name='s_browse' kind='recu_01' att='type,id,pid,desc,speed,stype,sid,spid' status=''>\r\n  <type   no='1' table='' col='' src='' />\r\n  <type2  no='2' table='' col='' src='' />\r\n  <type3  no='3' table='' col='' src='' />\r\n  <type4  no='4' table='' col='' src='' />\r\n  <type5  no='5' table='' col='' src='' />\r\n  <type6  no='6' table='' col='' src='' />\r\n  <id     no='7' table='' col='' src='' />\r\n  <pid    no='8' table='' col='' src='' />\r\n  <desc   no='9' table='' col='' src='' />\r\n  <speed  no='10' table='' col='' src=''/>\r\n  <stype  no='11' table='' col='' src=''/>\r\n  <sid    no='12' table='' col='' src=''/>\r\n  <spid   no='13' table='' col='' src=''/>\r\n</tree>", LoadOptions.None);
			xelement.SetAttributeValue("ver", info.Ver);
			xelement.SetAttributeValue("name", name);
			foreach (XElement xelement2 in xelement.Elements())
			{
				xelement2.SetAttributeValue("src", info.Env);
			}
			return new SpecTreeNode(info.Key, xelement);
		}

		// Token: 0x060006CF RID: 1743 RVA: 0x0001EAEC File Offset: 0x0001CCEC
		public static SpecTreeNode Create(PackageKey key, XElement source)
		{
			if (source == null)
			{
				return null;
			}
			return new SpecTreeNode(key, source);
		}

		// Token: 0x170001EF RID: 495
		// (get) Token: 0x060006D0 RID: 1744 RVA: 0x0001EAFA File Offset: 0x0001CCFA
		public string Error
		{
			get
			{
				return string.Empty;
			}
		}

		// Token: 0x170001F0 RID: 496
		public string this[string columnName]
		{
			get
			{
				return "未實作完成";
			}
		}
	}
}
